FROM alpine:latest AS api
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=rmc-builder /go/src/github.com/miragespace/rmc/bin/api /root/
CMD ["/root/api"]