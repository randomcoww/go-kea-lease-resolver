FROM alpine:latest

COPY go-kea-lease-resolver /
ENTRYPOINT ["/go-kea-lease-resolver"]
