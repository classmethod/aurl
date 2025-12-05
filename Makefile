GOFILES    := $(shell find . -name '*.go' -not -path './vendor/*')
NAME       := aurl


.DEFAULT_GOAL := bin/$(NAME)
bin/$(NAME): $(GOFILES)
	go build -o bin/$(NAME)

.PHONY: clean
clean:
	rm -rf vendor
	rm -rf bin/*
	rm -rf dist/*

.PHONY: deps
deps:
	go mod download

.PHONY: build
build:
	go build $(LDFLAGS) -o bin/$(NAME)

.PHONY: test
test:
	go test -v $(GOPACKAGES)

.PHONY: gorelease
gorelease:
	goreleaser release --snapshot --clean
