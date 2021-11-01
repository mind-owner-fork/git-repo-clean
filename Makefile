TARGET := git-clean-repo

VERSION := 0.0.1

GO_VERSION := $(subst go version go,,$(shell go version))

BUILD_VERSION := $(shell git describe --tags --always)

GO_LDFLAGS := -X 'main.GoVersion=$(GO_VERSION)' -X 'main.BuildVersion=$(BUILD_VERSION)'
GOFLAGS := -ldflags "$(GO_LDFLAGS)"


.PHONY: all
all:
	go build -o bin/${TARGET} $(GOFLAGS)


.PHONY: release
release:

release:
	mkdir -p releases
	$(call RELEASE_BUILD,linux,amd64)
	$(call RELEASE_BUILD,linux,386)
	$(call RELEASE_BUILD,darwin,amd64)
	$(call RELEASE_BUILD,windows,amd64,.exe)
	$(call RELEASE_BUILD,windows,386,.exe)

RELEASE_BUILD = GOOS=$(1) GOARCH=$(2) \
	go build \
	 $(GOFLAGS) \
	-o ./releases/git-clean-repo-$(1)-$(2)$(3)

.PHONY: clean
clean:
	rm -f bin/git-clean-repo
	rm -rf releases/*
