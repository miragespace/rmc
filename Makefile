COMMIT_HASH := $(shell git rev-parse --short HEAD)

all: build_api build_worker

proto:
	docker build -t protogen -f Dockerfile.protogen .
	rm -rf spec/protocol/*.pb.go
	docker run -v `pwd`:/proto protogen --go_opt=paths=source_relative --go_out=. spec/protocol/*.proto

cockroach:
	docker exec -ti rmc-crdb cockroach sql -d rmc --insecure

builder:
	docker build -t rmc-builder -f Dockerfile.builder .

image:
	docker build -t rachel.sh/miragespace/rmc-api ./cmd/api
	docker build -t rachel.sh/miragespace/rmc-task ./cmd/task
	docker build -t rachel.sh/miragespace/rmc-worker ./cmd/worker

multi: builder image
	docker-compose up -f docker-compose-multi.yml --remove-orphans
