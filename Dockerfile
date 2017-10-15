FROM debian:sid

COPY go-kea-lease-resolver /
ENTRYPOINT ["/go-kea-lease-resolver"]
