COMMIT_HASH := $(shell git rev-parse --short HEAD)

all: build_api build_worker

proto:
	docker build -t protogen -f Dockerfile.protogen .
	rm -rf spec/protocol/*.pb.go
	docker run -v `pwd`:/proto protogen --go_opt=paths=source_relative --go_out=. spec/protocol/*.proto

build_api:
	go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/api ./cmd/api
	go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/task ./cmd/task

build_worker:
	go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/worker ./cmd/worker
	GOOS=windows go build -ldflags "-X 'main.Version=$(COMMIT_HASH)'" -o bin/worker.exe ./cmd/worker

psql:
	docker exec -ti rmc-postgres psql -U rmc