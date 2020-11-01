FROM alpine:latest AS worker
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=rmc-builder /go/src/github.com/zllovesuki/rmc/bin/worker /root/
CMD ["/root/worker"]