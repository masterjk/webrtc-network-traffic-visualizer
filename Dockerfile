ARG VERSION=0.0.0

FROM golang:1.19.0-alpine as builder
WORKDIR /go/src/github.com/masterjk/webrtc-network-traffic-visualizer
COPY . $WORKDIR
RUN GOOS=linux CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=${VERSION}" -v -o visualizer ./cmd/server

FROM alpine:3.16.2
COPY --from=builder /go/src/github.com/masterjk/webrtc-network-traffic-visualizer/visualizer /visualizer
COPY web /web
EXPOSE 8080
CMD ["/visualizer"]
