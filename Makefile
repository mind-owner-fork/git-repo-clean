TARGET := git-repo-clean

VERSION := 1.3.2

GO_VERSION := $(subst go version go,,$(shell go version))

BUILD_VERSION := $(shell git describe --tags --always)

GO_LDFLAGS := -X 'main.GoVersion=$(GO_VERSION)' -X 'main.BuildVersion=$(BUILD_VERSION)'
GOFLAGS := -ldflags "$(GO_LDFLAGS)"


.PHONY: all
all:
	go build -o bin/${TARGET} $(GOFLAGS)


.PHONY: release
release: Linux MacOS Windows


Linux: Linux-64 Linux-32
Linux-64:
	mkdir -p releases/$(VERSION)/Linux-64/
	cp -r docs releases/$(VERSION)/Linux-64/
	cp LICENSE releases/$(VERSION)/Linux-64/
	cp README.md releases/$(VERSION)/Linux-64/
	$(call RELEASE_BUILD,linux,amd64)
	cp ./releases/git-repo-clean releases/$(VERSION)/Linux-64/
	tar -cf releases/git-repo-clean-$(VERSION)-Linux-64.tar releases/$(VERSION)/Linux-64/
	rm releases/git-repo-clean
	rm -rf releases/$(VERSION)/Linux-64

Linux-32:
	mkdir -p releases/$(VERSION)/Linux-32/
	cp -r docs releases/$(VERSION)/Linux-32/
	cp LICENSE releases/$(VERSION)/Linux-32/
	cp README.md releases/$(VERSION)/Linux-32/
	$(call RELEASE_BUILD,linux,386)
	cp ./releases/git-repo-clean releases/$(VERSION)/Linux-32/
	tar -cf releases/git-repo-clean-$(VERSION)-Linux-32.tar releases/$(VERSION)/Linux-32/
	rm releases/git-repo-clean
	rm -rf releases/$(VERSION)/Linux-32/

MacOS: macOS-64
macOS-64:
	mkdir -p releases/$(VERSION)/macOS-64/
	cp -r docs releases/$(VERSION)/macOS-64/
	cp LICENSE releases/$(VERSION)/macOS-64/
	cp README.md releases/$(VERSION)/macOS-64/
	$(call RELEASE_BUILD,darwin,amd64)
	cp ./releases/git-repo-clean releases/$(VERSION)/macOS-64/
	tar -cf releases/git-repo-clean-$(VERSION)-macOS-64.tar releases/$(VERSION)/macOS-64/
	rm releases/git-repo-clean
	rm -rf releases/$(VERSION)/macOS-64/

Windows: Windows-64 Windows-32
Windows-64:
	mkdir -p releases/$(VERSION)/Windows-64/
	cp -r docs releases/$(VERSION)/Windows-64/
	cp LICENSE releases/$(VERSION)/Windows-64/
	cp README.md releases/$(VERSION)/Windows-64/
	$(call RELEASE_BUILD,windows,amd64,.exe)
	cp ./releases/git-repo-clean.exe releases/$(VERSION)/Windows-64/
	zip -r releases/git-repo-clean-$(VERSION)-Windows-64.zip releases/$(VERSION)/Windows-64/
	rm releases/git-repo-clean.exe
	rm -rf releases/$(VERSION)/Windows-64/

Windows-32:
	mkdir -p releases/$(VERSION)/Windows-32/
	cp -r docs releases/$(VERSION)/Windows-32/
	cp LICENSE releases/$(VERSION)/Windows-32/
	cp README.md releases/$(VERSION)/Windows-32/
	$(call RELEASE_BUILD,windows,386,.exe)
	cp ./releases/git-repo-clean.exe releases/$(VERSION)/Windows-32/
	zip -r releases/git-repo-clean-$(VERSION)-Windows-32.zip releases/$(VERSION)/Windows-32/
	rm releases/git-repo-clean.exe
	rm -rf releases/$(VERSION)/Windows-32/

RELEASE_BUILD = GOOS=$(1) GOARCH=$(2) \
	go build \
	 $(GOFLAGS) \
	-o ./releases/git-repo-clean$(3)
.PHONY: clean
clean:
	rm -f bin/git-repo-clean
	rm -rf releases/*

.PHONY: install
install:
	cp bin/git-repo-clean $(shell git --exec-path)
