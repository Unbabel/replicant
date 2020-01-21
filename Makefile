GOLDFLAGS += -w -extldflags "-static"
GOLDFLAGS += -X main.Version=$(shell git describe)
GOLDFLAGS += -X main.GitCommit=$(shell git rev-parse HEAD)
GOLDFLAGS += -X main.BuildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GOFLAGS = -mod=vendor -ldflags "$(GOLDFLAGS)"

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o replicant cmd/replicant/*.go

test:
	go vet ./...
	go test -race -cover ./...

