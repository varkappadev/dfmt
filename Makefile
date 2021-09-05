.SUFFIXES:
.PHONY: tidy test release

BINARY:=$(shell go list | head -n 1 | xargs basename)
VERSION:=$(shell git describe --tags --first-parent --long --dirty=+dev 2>/dev/null || echo 0.0.1)
TAG:=$(shell git describe --first-parent --long --dirty=+dev --always)

${BINARY}: *.go Makefile go.sum go.mod
	go build -buildmode=pie -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.tag=${TAG}' -X 'main.appName=${BINARY}'" -o "${BINARY}"

release: ${BINARY} test tidy 

tidy:
	go fmt
	go mod tidy -v
	go mod verify
	go vet

test:
	go test -cover -test.v

coverage.html: *.go Makefile go.sum go.mod
	go test -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
