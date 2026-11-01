.PHONY: run test vet fmt build clean

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

clean:
	rm -rf bin
