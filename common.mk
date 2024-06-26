# Set an output prefix, which is the local directory if not specified
PREFIX ?= $(shell pwd)

# Setup name variables for the package/tool
NAME ?= $(shell basename $(PWD))
IMAGENAME ?= $(shell echo $(NAME) | tr '[:upper:]' '[:lower:]')
PKG := github.com/mchudgins/go

# set the docker repo name
REPO := mchudgins

# Set any default go build tags
BUILDTAGS :=

# Set the build dir, where built cross-compiled binaries will be output
BUILDDIR ?= ${PREFIX}/.build

# Populate version variables
# Add to compile time flags
VERSION := $(shell cat VERSION.txt)
GITCOMMIT := $(shell git rev-parse --short HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
ifneq ($(GITUNTRACKEDCHANGES),)
	GITCOMMIT := $(GITCOMMIT)-dirty
endif
GO ?= GO111MODULE=on go
GOFLAGS ?= -mod vendor
DEPS = $(shell $(GO) list ${GOFLAGS} -f '{{join .Deps  "\n"}}' $(MAIN) | grep -v vendor | grep ecco_cp)
CTIMEVAR=-X $(PKG)/version.GITCOMMIT=$(GITCOMMIT) -X $(PKG)/version.VERSION=$(VERSION)
GO_LDFLAGS ?= -ldflags "-w $(CTIMEVAR)"
GO_LDFLAGS_STATIC ?= -ldflags "-w $(CTIMEVAR) -extldflags -static"

# List the GOOS and GOARCH to build
#GOOSARCHES = darwin/amd64 darwin/386 freebsd/amd64 freebsd/386 linux/arm linux/arm64 linux/amd64 linux/386 windows/amd64 windows/386
GOOSARCHES ?= linux/amd64

.PHONY: build
build: $(NAME) ## Builds a dynamic executable or package

$(NAME): *.go VERSION.txt
	@echo "+ $@"
	$(GO) build ${GOFLAGS} -tags "$(BUILDTAGS)" ${GO_LDFLAGS} -o $(NAME) .

.PHONY: static
static: ## Builds a static executable
	@echo "+ $@"
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) \
				-installsuffix cgo \
				-tags "$(BUILDTAGS) static_build" \
				${GO_LDFLAGS_STATIC} -o $(BUILDDIR)/$(NAME) .

all: clean build fmt lint test staticcheck vet install docker ## Runs a clean, build, fmt, lint, test, staticcheck, vet, docker and install

deps:
	@$(GO) list ${GOFLAGS} -f '{{join .Deps  "\n"}}' ${NAME}.go | grep -v vendor | grep ecco_cp

.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed
	@echo "+ $@"
	@goimports -l ../.. | grep -v '.pb.go:' | grep -v vendor | tee /dev/stderr

.PHONY: lint
lint: ## Verifies `golint` passes
	@echo "+ $@"
#	@CGO_ENABLED=0 GO111MODULE=on golint ./... | grep -v '.pb.go:' | grep -v vendor | tee /dev/stderr
	sudo docker run --rm -e CGO_ENABLED=0 -e GO111MODULE=on -v $(PWD):/app -w /app golangci/golangci-lint:v1.51-alpine golangci-lint run -v

.PHONY: test
test: ## Runs the go tests
	@echo "+ $@"
	@CGO_ENABLED=0 $(GO) test  ${GOFLAGS} -cover -v -installsuffix cgo -tags "$(BUILDTAGS) cgo" $(shell $(GO) list ../../... | grep -v vendor)

.PHONY: vet
vet: ## Verifies `go vet` passes
	@echo "+ $@"
	@$(GO) vet ${GOFLAGS} $(shell $(GO) list ../../... | grep -v vendor) | grep -v '.pb.go:' | tee /dev/stderr

.PHONY: staticcheck
staticcheck: ## Verifies `staticcheck` passes
	@echo "+ $@"
	@CGO_ENABLED=0 staticcheck $(shell $(GO) list ../../... | grep -v vendor) | grep -v '.pb.go:' | grep -v '/usr/local/go' | tee /dev/stderr

.PHONY: cover
cover: ## Runs go test with coverage
	@echo "" > coverage.txt
	@for d in $(shell $(GO) list ../../... | grep -v vendor); do \
		$(GO) test ${GOFLAGS} -race -coverprofile=profile.out -covermode=atomic "$$d"; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi; \
	done;

.PHONY: install
install: ## Installs the executable or package
	@echo "+ $@"
	$(GO) install ${GOFLAGS} -tags "$(BUILDTAGS)" ${GO_LDFLAGS} .

define buildpretty
mkdir -p $(BUILDDIR)/$(1)/$(2);
GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 $(GO) build ${GOFLAGS} \
	 -o $(BUILDDIR)/$(1)/$(2)/$(NAME) \
	 -tags "$(BUILDTAGS) static_build netgo" \
	 -installsuffix netgo ${GO_LDFLAGS_STATIC} .;
md5sum $(BUILDDIR)/$(1)/$(2)/$(NAME) > $(BUILDDIR)/$(1)/$(2)/$(NAME).md5;
sha256sum $(BUILDDIR)/$(1)/$(2)/$(NAME) > $(BUILDDIR)/$(1)/$(2)/$(NAME).sha256;
endef

.PHONY: cross
cross: *.go VERSION.txt ## Builds the cross-compiled binaries, creating a clean directory structure (eg. GOOS/GOARCH/binary)
	@echo "+ $@"
	$(foreach GOOSARCH,$(GOOSARCHES), $(call buildpretty,$(subst /,,$(dir $(GOOSARCH))),$(notdir $(GOOSARCH))))

define buildrelease
GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 $(GO) build ${GOFLAGS} \
	 -o $(BUILDDIR)/$(NAME)-$(1)-$(2) \
	 -tags "$(BUILDTAGS) static_build netgo" \
	 -installsuffix netgo ${GO_LDFLAGS_STATIC} .;
md5sum $(BUILDDIR)/$(NAME)-$(1)-$(2) > $(BUILDDIR)/$(NAME)-$(1)-$(2).md5;
sha256sum $(BUILDDIR)/$(NAME)-$(1)-$(2) > $(BUILDDIR)/$(NAME)-$(1)-$(2).sha256;
endef

.PHONY: release
release: *.go VERSION.txt ## Builds the cross-compiled binaries, naming them in such a way for release (eg. binary-GOOS-GOARCH)
	@echo "+ $@"
	$(foreach GOOSARCH,$(GOOSARCHES), $(call buildrelease,$(subst /,,$(dir $(GOOSARCH))),$(notdir $(GOOSARCH))))

.PHONY: bump-version
BUMP := minor
bump-version:  ## Bump the version in the version file. Set BUMP to [ patch | major | minor ]
	@go get -u github.com/jessfraz/junk/sembump # update sembump tool
	@$(shell if [ ! -f VERSION.txt ]; then echo "v0.0.0" >VERSION.txt; fi)
	$(eval NEW_VERSION = $(shell sembump --kind $(BUMP) $(VERSION)))
	@echo "Bumping VERSION.txt from $(VERSION) to $(NEW_VERSION)"
	echo $(NEW_VERSION) > VERSION.txt
	@echo "Updating links to download binaries in README.md"
	sed -i s/$(VERSION)/$(NEW_VERSION)/g README.md
	git add VERSION.txt README.md
	git commit -vsam "Bump $(NAME) version to $(NEW_VERSION)"
	@echo "Run make tag to create and push the tag for new version $(NEW_VERSION)"

.PHONY: tag
tag: ## Create a new git tag to prepare to build a release
	git tag -sa $(NAME)-$(VERSION) -m "$(NAME)-$(VERSION)"
	@echo "Run git push origin $(NAME)-$(VERSION) to push your new tag to GitHub and trigger a release build."

.PHONY: docker
docker: docker/Dockerfile release ## create a container from the linux/amd64 image
	@echo "+ $@"
	@cp $(BUILDDIR)/$(NAME)-linux-amd64 docker/$(NAME)
	sudo docker build -t $(REPO)/$(IMAGENAME):$(VERSION) docker

.PHONY: AUTHORS
AUTHORS:
	@$(file >$@,# This file lists all individuals having contributed content to the repository.)
	@$(file >>$@,# For how it is generated, see `make AUTHORS`.)
	@echo "$(shell git log --format='\n%aN <%aE>' | LC_ALL=C.UTF-8 sort -uf | ../../bin/collapse-authors)" >> $@

.PHONY: clean
clean: ## Cleanup any build binaries or packages
	@echo "+ $@"
	$(RM) $(NAME)
	#$(RM) -r $(BUILDDIR)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
