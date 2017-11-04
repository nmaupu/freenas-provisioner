BIN=bin

all: build

fmt:
	go fmt ./...

build: bin
	env CGO_ENABLED=0 go build -o $(BIN)/freenas-provisioner

install:
	env CGO_ENABLED=0 go install

clean:
	go clean -i
	rm -rf $(BIN)

bin:
	mkdir -p $(BIN)

.PHONY: fmt install clean test all release
