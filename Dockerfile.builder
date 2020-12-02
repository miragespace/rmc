FROM golang:1.15.5-alpine

RUN apk add git

WORKDIR /go/src/github.com/miragespace/rmc
COPY .git .
ADD . /go/src/github.com/miragespace/rmc

ENV CGO_ENABLED=0

RUN GIT_COMMIT=$(git rev-parse --short HEAD) && \
    go build -ldflags "-X 'main.Version=$GIT_COMMIT'" -o bin/api ./cmd/api
RUN GIT_COMMIT=$(git rev-parse --short HEAD) && \
    go build -ldflags "-X 'main.Version=$GIT_COMMIT'" -o bin/task ./cmd/task
RUN GIT_COMMIT=$(git rev-parse --short HEAD) && \
    go build -ldflags "-X 'main.Version=$GIT_COMMIT'" -o bin/worker ./cmd/worker
