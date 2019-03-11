package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"flag"
	"github.com/vanng822/go-solr/solr"
)

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
	resource            = "/media"
	contentEncodingJSON = "contentEncJson"
	attributesJSON      = "attributesJson"
)

func main() {
	log.Print("Content Manager started.")
	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router()))
}

func router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Public routes
	r.Get("/", index)

	r.Route(resource, func(r chi.Router) {
		// Subrouters:
		r.Get("/", rootIndex)

		r.Route("/{contentId}", func(r chi.Router) {
			r.Get("/", fetchContent)
		})
	})

	return r
}

func index(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Content Media Server!"))
}

func rootIndex(w http.ResponseWriter, r *http.Request) {
	http.Error(w, fmt.Sprintf("Missing contentId, invoke /%s/<contentId>", resource), http.StatusBadRequest)
}

func fetchContent(w http.ResponseWriter, r *http.Request) {
	contentID := chi.URLParam(r, "contentId")

	solrEndpoint := os.Getenv("SOLR_ENDPOINT")
	if solrEndpoint == "" {
		http.Error(w, "SOLR_ENDPOINT is not defined in environment", http.StatusInternalServerError)
		return
	}

	log.Printf("Searching SOLR for contentId: %s", contentID)

	si, err := solr.NewSolrInterface(solrEndpoint, "catcollection")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := solr.NewQuery()
	query.Q(fmt.Sprintf("id:%s", contentID))

	s := si.Search(query)
	log.Println("s: >> %s", s)

	result, resultError := s.Result(nil)
	if resultError != nil {
		http.Error(w, resultError.Error(), http.StatusInternalServerError)
		return
	}

	if result == nil || result.Results == nil || len(result.Results.Docs) == 0 {
		http.Error(w, "Try again later", http.StatusNotFound)
		return
	}

	doc := result.Results.Docs[0]

	if !doc.Has(contentEncodingJSON) || !doc.Has(attributesJSON) {
		http.Error(w, "Invalid Content", http.StatusInternalServerError)
	}

	rawEncodings := doc.Get(contentEncodingJSON).([]interface{})
	encodings := make([]Encoding, len(rawEncodings))

	for i, val := range rawEncodings {
		var enc Encoding
		s := val.(string)
		err := json.Unmarshal([]byte(s), &enc)
		if err == nil {
			encodings[i] = enc
		}
	}

	rawAttributes := doc.Get(attributesJSON).([]interface{})
	attributes := make(map[string]interface{}, len(rawAttributes))
	for _, val := range rawAttributes {
		var att Attribute
		s := val.(string)
		err := json.Unmarshal([]byte(s), &att)
		if err == nil {
			attributes[att.Name] = att.Value
		}
	}

	media := &Media{
		encodings,
		attributes,
	}
	json, err := json.Marshal(media)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(json)
}
