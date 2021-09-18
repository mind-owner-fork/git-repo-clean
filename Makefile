TARGET := git-clean-repo

VERSION := 0.0.1

GO_VERSION := $(subst go version go,,$(shell go version))

BUILD_VERSION := $(shell git describe --tags --always)

GO_LDFLAGS := -X 'main.GoVersion=$(GO_VERSION)' -X 'main.BuildVersion=$(BUILD_VERSION)'
GOFLAGS := -ldflags "$(GO_LDFLAGS)"


.PHONY: all
all:
	go build -o ${TARGET} $(GOFLAGS)

.PHONY: clean
clean:
	rm -f git-clean-repo
