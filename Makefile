PERCEIVERS = $(shell ls cmd)

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

OUTDIR=_output
LOCAL_TARGET=local

CURRENT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: all clean test push test ${PERCEIVERS} container local

all: build

build: ${OUTDIR} $(PERCEIVERS)

${LOCAL_TARGET}: ${OUTDIR} $(PERCEIVERS)

$(PERCEIVERS):
ifeq ($(MAKECMDGOALS),${LOCAL_TARGET})
	cd cmd/$@; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@
else
	docker run --rm -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 -v "${CURRENT_DIR}":/go/src/github.com/blackducksoftware/perceivers -w /go/src/github.com/blackducksoftware/perceivers/cmd/$@ golang:1.9 go build -o $@
endif
	cp cmd/$@/$@ ${OUTDIR}

container: $(PERCEIVERS)
	$(foreach p,${PERCEIVERS},cd ${CURRENT_DIR}/cmd/$p; docker build -t $(REGISTRY)/$(PREFIX)${p} .;)

push: container
	$(foreach p,${PERCEIVERS},$(PREFIX_CMD) docker $(DOCKER_OPTS) push $(REGISTRY)/$(PREFIX)${p}:latest;)

test:
	docker run --rm -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 -v "${CURRENT_DIR}":/go/src/github.com/blackducksoftware/perceivers -w /go/src/github.com/blackducksoftware/perceivers golang:1.9 go test ./pkg/...

clean:
	rm -rf ${OUTDIR}
	$(foreach p,${PERCEIVERS},rm -f cmd/$p/$p;)

${OUTDIR}:
	mkdir -p ${OUTDIR}

lint:
	./hack/verify-gofmt.sh
	./hack/verify-golint.sh
	./hack/verify-govet.sh
