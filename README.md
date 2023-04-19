[![Docker Image CI](https://github.com/masterjk/webrtc-network-traffic-visualizer/actions/workflows/docker-image.yml/badge.svg)](https://github.com/masterjk/webrtc-network-traffic-visualizer/actions/workflows/docker-image.yml)

# webrtc-network-traffic-visualizer

Web application to test WebRTC network traffic conditions over various network deployments.

## Usage

```
$ docker stop visualizer
$ docker rm visualizer
$ docker run --name visualizer -d -p 8080:8080 docker.io/josephkiok/webrtc-network-traffic-visualizer:latest
```
