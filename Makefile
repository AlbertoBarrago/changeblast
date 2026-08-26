.PHONY: build test vet fmt lint man man-check

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/AlbertoBarrago/changeblast/cmd.version=$(VERSION) \
           -X github.com/AlbertoBarrago/changeblast/cmd.commit=$(COMMIT) \
           -X github.com/AlbertoBarrago/changeblast/cmd.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o blast .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

lint: fmt vet test

man:
	go run ./tools/gendocs docs

man-check: man
	git diff --exit-code docs/*.1 || (echo "docs/*.1 is out of date — run 'make man' and commit the result" && exit 1)
