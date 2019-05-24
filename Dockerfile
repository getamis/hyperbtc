# Build hypereth in a stock Go builder container
FROM golang:1.12-alpine3.9 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . $GOPATH/src/github.com/getamis/hyperbtc
RUN mkdir /hyperbtc
RUN cd $GOPATH/src/github.com/getamis/hyperbtc \
&& go install ./cmd/... \
&& mv -v /go/bin/* /hyperbtc

# Pull hypereth into a second stage deploy alpine container
FROM alpine:3.9

RUN apk add --no-cache ca-certificates
COPY --from=builder /hyperbtc /usr/local/bin/
