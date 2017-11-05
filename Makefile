BIN=bin

IMAGE_NAME=freenas-provisioner
IMAGE_VERSION=0.7
REMOTE_NAME=$(DOCKER_ID_USER)/$(IMAGE_NAME)

all: $(BIN)/freenas-provisioner

fmt:
	go fmt ./...

image: $(BIN)/freenas-provisioner check-docker-hub
	docker build -t $(IMAGE_NAME):$(IMAGE_VERSION) -f Dockerfile.scratch .

tag: image
	docker tag $(IMAGE_NAME):$(IMAGE_VERSION) $(REMOTE_NAME):$(IMAGE_VERSION)

push: tag
	docker push $(REMOTE_NAME):$(IMAGE_VERSION)

vendor:
	glide install -v

$(BIN)/freenas-provisioner: vendor $(BIN) $(shell find . -name "*.go")
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o $(BIN)/freenas-provisioner .

install:
	env CGO_ENABLED=0 go install

clean:
	go clean -i
	rm -rf $(BIN)
	rm -rf vendor

$(BIN):
	mkdir -p $(BIN)

check-docker-hub:
ifndef DOCKER_ID_USER
	$(error ERROR! DOCKER_ID_USER environment variable must be defined)
endif

.PHONY: fmt install clean test all image
