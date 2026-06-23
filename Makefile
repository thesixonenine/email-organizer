.PHONY: build run test clean coverage lint deps

APP_NAME = organizer

build:
	go build -o bin/$(APP_NAME) ./cmd/organizer/

run: build
	./bin/$(APP_NAME) -config configs/config.yaml

test:
	go test ./... -v -count=1

test-race:
	go test ./... -race -count=1

coverage:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html data/

lint:
	go vet ./...

deps:
	go mod tidy