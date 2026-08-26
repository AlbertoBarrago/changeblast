.PHONY: build test vet fmt lint man

build:
	go build -o blast .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

lint: fmt vet test

man:
	@echo "man page generation (cobra/doc) not yet wired up — planned once command surface stabilizes"
