ifndef REGISTRY
REGISTRY=gcr.io/gke-verification
endif

ifdef IMAGE_PREFIX
PREFIX="$(IMAGE_PREFIX)-"
endif

ifneq (, $(findstring gcr.io,$(REGISTRY))) 
PREFIX_CMD="gcloud"
DOCKER_OPTS="--"
endif

.PHONY: clean test

all: build

build:
	mkdir -p _output
	cd cmd && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o image-perceiver image-perceiver.go
	cp cmd/image-perceiver _output

container: build
	cd cmd && docker build -t $(REGISTRY)/$(PREFIX)image-perceiver .

push: container
	$(PREFIX_CMD) docker $(DOCKER_OPTS) push $(REGISTRY)/$(PREFIX)image-perceiver:latest

test:
	go test ./pkg/...

clean:
	rm -rf _output cmd/image-perceiver
