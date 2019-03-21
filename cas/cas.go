package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fsnotify/fsnotify"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
	"github.com/spf13/viper"

	"flag"
)

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *errorResponse) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Message)
}

// Media represents a playback item.
type Media struct {
	Encodings  []Encoding             `json:"encodings"`
	Attributes map[string]interface{} `json:"attributes"`
}

// Attribute represents custom attributes for given media.
type Attribute struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	id    int64
	Type  int
}

// Encoding represents specifics of the playback item (i.e. DRM, Package type etc).
type Encoding struct {
	Channel           string `json:"channel,omitempty"`
	URI               string `json:"uri"`
	MetaID            string `json:"metaId"`
	EncodingProfileID string `json:"encodingProfileId"`
	DrmVersion        string `json:"drmVersion,omitempty"`
	DrmID             string `json:"drmId,omitempty"`
	Status            int    `json:"status"`
	CreatedDatetime   int64  `json:"created"`
	UpdatedDatetime   int64  `json:"updated"`
}

const (
	defaultDrmID = "config.app.DEFAULT_DRM_ID"
)

var (
	resource  string
	tokenAuth *jwtauth.JWTAuth
	config    *configuration
)

// Represents Viper configuration.
type configuration struct {
	v *viper.Viper
}

func setupConfig() *configuration {
	c := configuration{
		v: viper.New(),
	}
	c.v.SetConfigName("cas-app-config")
	c.v.SetConfigType("yaml")
	c.v.SetDefault(defaultDrmID, "6")
	c.v.AutomaticEnv()
	c.v.AddConfigPath(".")
	c.v.AddConfigPath(os.Getenv("CONFIG_PATH"))

	c.v.SetTypeByDefaultValue(true)
	err := c.v.ReadInConfig()
	if _, ok := err.(*os.PathError); ok {
		log.Printf("no config file found. Using default values")
	} else if err != nil { // Handle other errors that occurred while reading the config file
		log.Printf("fatal error while reading the config file: %s", err)
	}
	c.v.WatchConfig()
	c.v.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed, file: %s", e.Name)
		log.Println("default DRM Id from config: %s", c.GetDefaultDrmID())
	})
	return &c
}

// GetLogLevel returns the log level
func (c *configuration) GetDefaultDrmID() string {
	return c.v.GetString(defaultDrmID)
}

func init() {
	tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)

	// For debugging/example purposes, we generate and print
	// a sample jwt token with claims `user_id:123` here:
	_, tokenString, _ := tokenAuth.Encode(jwt.MapClaims{"user_id": 123})
	log.Print("DEBUG: a sample jwt is %s\n\n", tokenString)
}

func main() {
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	config = setupConfig()

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router()))
}

func router() http.Handler {
	log.Printf("Log Level from config: %s", config.GetDefaultDrmID())
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	resource = os.Getenv("RESOURCE")
	if resource == "" {
		resource = "NOT SPECIFIED"
	}

	// Public routes
	r.Get("/", index)

	// Protected routes
	r.Route(resource, func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)

		// Subrouters:
		r.Get("/", rootIndex)

		r.Route("/{contentID}", func(r chi.Router) {
			r.Get("/", authorize)
		})
	})

	return r
}

func index(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Content Authorization Server!"))
}

func rootIndex(w http.ResponseWriter, r *http.Request) {
	http.Error(w, fmt.Sprintf("Missing contentId, invoke /%s/<contentId>", resource), http.StatusBadRequest)
}

func authorize(w http.ResponseWriter, r *http.Request) {
	contentID := chi.URLParam(r, "contentID")

	contentManagerEndpoint := os.Getenv("CONTENT_MANAGER_ENDPOINT")
	if contentManagerEndpoint == "" {
		http.Error(w, fmt.Sprintf("CONTENT_MANAGER_ENDPOINT is not defined in environment", resource), http.StatusInternalServerError)
		return
	}

	media, err := fetchMedia(contentID, contentManagerEndpoint)
	if err != nil {
		writeError(w, 0, http.StatusText(http.StatusInternalServerError))
		return
	}

	filteredEncodings := filter(media.Encodings, validEncoding)
	if len(filteredEncodings) == 0 {
		writeError(w, 306, "No matching encodings found")
		return
	}

	out := fmt.Sprintf("%s | %s", os.Getenv("APP_VERSION"), filteredEncodings[0].URI)
	w.Write([]byte(out))
}

func fetchMedia(contentID string, cmsURI string) (Media, error) {
	path := fmt.Sprintf("/media/%s", contentID)
	u := url.URL{
		Scheme: "http",
		Host:   cmsURI,
		Path:   path,
	}

	log.Printf("Fetch content from %s", u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		return Media{}, errors.New(http.StatusText(http.StatusInternalServerError))
	}

	if resp.StatusCode != 200 {
		return Media{}, errors.New(http.StatusText(resp.StatusCode))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Media{}, errors.New(http.StatusText(resp.StatusCode))
	}

	var media Media
	if err = json.Unmarshal([]byte(body), &media); err != nil {
		return Media{}, errors.New(http.StatusText(resp.StatusCode))
	}

	return media, nil
}

func writeError(w http.ResponseWriter, code int, msg string) {
	err := &errorResponse{
		code,
		msg,
	}
	json, _ := json.Marshal(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(json)
}

func writeErrorJSON(w http.ResponseWriter, err errorResponse) {
	json, _ := json.Marshal(err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(json)
}

func validEncoding(enc Encoding) bool {
	defaultDrmID := config.GetDefaultDrmID()
	return enc.DrmID == defaultDrmID
}

func filter(vs []Encoding, f func(Encoding) bool) []Encoding {
	vsf := make([]Encoding, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}
