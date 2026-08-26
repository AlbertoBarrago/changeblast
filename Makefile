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
	go run ./tools/gendocs docs

man-check: man
	git diff --exit-code docs/*.1 || (echo "docs/*.1 is out of date — run 'make man' and commit the result" && exit 1)
