SUBDIRS = pod image

.PHONY: $(SUBDIRS) clean

all: build

build: outputdir $(SUBDIRS)
	cp */_output/* _output

outputdir:
	mkdir -p _output

container: $(SUBDIRS)

push: $(SUBDIRS)

test: $(SUBDIRS)
	go test ./pkg/...

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

clean: $(SUBDIRS)
	rm -rf _output
