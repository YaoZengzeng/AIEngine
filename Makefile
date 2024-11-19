IMAGE_PREFIX ?= ghcr.io/yaozengzeng/
APP_NAME ?= aiengine
TAG ?= latest

.PHONY: docker-buildx
docker-buildx:
	docker buildx build --no-cache . -t $(IMAGE_PREFIX)$(APP_NAME):$(TAG) --build-arg GO_LDFLAGS="$(GO_LDFLAGS)" --load
