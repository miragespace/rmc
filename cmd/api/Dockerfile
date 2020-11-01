FROM alpine:latest AS api
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=rmc-builder /go/src/github.com/zllovesuki/rmc/bin/api /root/
CMD ["/root/api"]