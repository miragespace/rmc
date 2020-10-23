COMMIT_HASH := $(shell git rev-parse --short HEAD)

all: build_api build_host

proto:
	docker build -t protogen -f Dockerfile.protogen .
	rm -rf spec/*.pb.go
	docker run -v `pwd`:/proto protogen --go_out=. spec/*.proto

build_api:
	go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/api ./cmd/api

build_host:
	go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/host ./cmd/host
	GOOS=windows go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/host.exe ./cmd/host