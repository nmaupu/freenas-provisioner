BIN=bin

IMAGE_NAME=freenas-provisioner
IMAGE_VERSION=2.3
REMOTE_NAME=$(DOCKER_ID_USER)/$(IMAGE_NAME)

all: build

fmt:
	go fmt ./...

tmp:
	mkdir -p tmp/

image: tmp check-docker-hub
	wget -O tmp/freenas-provisioner https://github.com/nmaupu/freenas-provisioner/releases/download/v$(IMAGE_VERSION)/freenas-provisioner_linux-amd64 && \
		chmod +x tmp/freenas-provisioner
	docker build -t $(IMAGE_NAME):$(IMAGE_VERSION) -f Dockerfile.alpine .

tag: image
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION) $(REMOTE_NAME):$(IMAGE_VERSION)

push: tag
	docker push $(REMOTE_NAME):$(IMAGE_VERSION)

vendor:
	glide install -v --strip-vcs

$(BIN)/freenas-provisioner build: vendor $(BIN) $(shell find . -name "*.go")
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $(BIN)/freenas-provisioner .

darwin: vendor $(BIN) $(shell find . -name "*.go")
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $(BIN)/freenas-provisioner-darwin .

freebsd: vendor $(BIN) $(shell find . -name "*.go")
	env CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $(BIN)/freenas-provisioner-freebsd .

clean:
	go clean -i
	rm -rf $(BIN)
	rm -rf tmp/
	rm -rf vendor

$(BIN):
	mkdir -p $(BIN)

check-docker-hub:
ifndef DOCKER_ID_USER
	$(error ERROR! DOCKER_ID_USER environment variable must be defined)
endif

.PHONY: all fmt clean image tag push
