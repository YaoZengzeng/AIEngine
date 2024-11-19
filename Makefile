IMAGE_PREFIX ?= ghcr.io/kmesh-net/
APP_NAME ?= aiengine
TAG ?= latest

.PHONY: docker-buildx
docker-buildx:
	docker buildx build . -t $(IMAGE_PREFIX)$(APP_NAME):$(TAG) --build-arg GO_LDFLAGS="$(GO_LDFLAGS)" --load
