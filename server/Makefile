.PHONY: build run test fmt vet tidy

GOOS_LINUX = linux
GOARCH_LINUX = amd64

build:
	CGO_ENABLED=0 GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_LINUX) go build -o bin/apppilot-server .

run:
	go run . serve

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy
