FROM alpine:latest AS task
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=rmc-builder /go/src/github.com/zllovesuki/rmc/bin/task /root/
CMD ["/root/task"]