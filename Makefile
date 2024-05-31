#
#

# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

# Set the build dir, where built cross-compiled binaries will be output
BUILDDIR := ${PREFIX}/.build

# build all the cmd's
.PHONY: all
all: $(wildcard cmd/*)
	@echo $^
	@for c in $^; do \
		cd $$c; \
		if [ -f Makefile ]; then \
			make clean release BUILDDIR=${BUILDDIR}; \
		else \
			echo No Makefile found in $$c; \
		fi; \
		cd - >/dev/null; \
	done;

.PHONY: unit-tests
unit-tests:
	GO111MODULE=on go test -cover -mod vendor ./...

.PHONY: lint
lint:
	#GO111MODULE=on gometalinter --vendor --deadline=2m --enable-gc --aggregate ./... | grep -v '/mod/'
	sudo docker run --rm -e CGO_ENABLED=0 -e GO111MODULE=on -v $(PWD):/app -w /app golangci/golangci-lint:v1.27.0 golangci-lint run -v

graph_dependencies:
	dep status -dot | dot -T png | display

