.PHONY: all build test coverage lint fmt-check vet clean install

all: lint test build

build:
	go build -o howdoi .

test:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

vet:
	go vet ./...

clean:
	rm -f howdoi coverage.out

install:
	go install .
