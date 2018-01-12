FROM golang:alpine as BUILD

WORKDIR /go/src/goapp/
COPY . .

RUN set -x \
  \
  && apk add --no-cache \
    git \
  \
  && go get -d ./... \
  && go build

FROM alpine:latest

COPY --from=BUILD /go/src/goapp/goapp /

ENTRYPOINT ["/goapp"]
