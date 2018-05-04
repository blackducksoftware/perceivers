PERCEIVERS = pod-perceiver image-perceiver

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
#BINARY=image-perceiver

CURRENT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: all clean test push test ${PERCEIVERS}

#.PHONY: $(SUBDIRS) clean build local_build container push test

all: build

build: ${OUTDIR} $(PERCEIVERS)

${LOCAL_TARGET}: ${OUTDIR} $(PERCEIVERS)

$(PERCEIVERS):
ifeq ($(MAKECMDGOALS),${LOCAL_TARGET})
	cd cmd/$@; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ $@.go
else
	docker run --rm -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 -v "${CURRENT_DIR}":/go/src/github.com/blackducksoftware/perceivers -w /go/src/github.com/blackducksoftware/perceivers/cmd/$@ golang go build -o $@ $@.go
endif
	cp cmd/$@/$@ ${OUTDIR}

container: $(PERCEIVERS)
	$(foreach p,${PERCEIVERS},cd ${CURRENT_DIR}/cmd/$p; docker build -t $(REGISTRY)/$(PREFIX)${p} .;)

push: container
	$(foreach p,${PERCEIVERS},$(PREFIX_CMD) docker $(DOCKER_OPTS) push $(REGISTRY)/$(PREFIX)${p}:latest;)

test:
	go test ./pkg/...

clean:
	rm -rf ${OUTDIR}
	$(foreach p,${PERCEIVERS},rm -f cmd/$p/$p;)

${OUTDIR}:
	mkdir -p ${OUTDIR}
