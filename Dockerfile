FROM golang:alpine as builder

ADD ./main.go /go/src/github.com/cirocosta/l7/main.go
ADD ./lib /go/src/github.com/cirocosta/l7/lib
ADD ./vendor /go/src/github.com/cirocosta/l7/vendor

WORKDIR /go/src/github.com/cirocosta/l7

RUN set -ex && \
  CGO_ENABLED=0 go build -tags netgo -v -a -ldflags '-extldflags "-static"' && \
  mv ./l7 /usr/bin/l7

FROM busybox
COPY --from=builder /usr/bin/l7 /usr/local/bin/l7

ENTRYPOINT [ "l7" ]

