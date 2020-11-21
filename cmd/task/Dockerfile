FROM debian:buster-slim AS api
WORKDIR /root/
COPY --from=rmc-builder /go/src/github.com/zllovesuki/rmc/bin/task /root/
CMD ["/root/task"]