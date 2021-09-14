TARGET := git-clean-repo

VERSION := 0.0.1

GO_VERSION := $(subst go version go,,$(shell go version))

GO_LDFLAGS := -X 'main.GoVersion=$(GO_VERSION)' -X 'main.BuildVersion=$(shell git describe --tags --always)'
GOFLAGS := -ldflags "$(GO_LDFLAGS)"


.PHONY: all
all:
	go build -o ${TARGET} $(GOFLAGS)

.PHONY: clean
clean:
	rm -f git-clean-repo
