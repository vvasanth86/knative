# Use the offical Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang as builder

# Copy local code to the container image.
WORKDIR /go/src/github.com/knative/vstb-lite/
COPY . .
# COPY ./models ../models

RUN go get .
# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN CGO_ENABLED=0 GOOS=linux go build -v -o cms

# Use a Docker multi-stage build to create a lean production image.
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
# FROM alpine
FROM gcr.io/distroless/base

# Copy the binary to the production image from the builder stage.
COPY --from=builder /go/src/github.com/knative/vstb-lite/cms /cms

# Service must listen to $PORT environment variable.
# This default value facilitates local development.
ENV PORT 8081

# Run the web service on container startup.
CMD ["/cms"]
