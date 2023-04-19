
IMAGE_NAME=josephkiok/webrtc-network-traffic-visualizer:latest

default: build

.PHONY:build
build:
	@docker build -t ${IMAGE_NAME} .
	@docker push ${IMAGE_NAME}
