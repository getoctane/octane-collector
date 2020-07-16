FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git ca-certificates
WORKDIR $GOPATH/src/github.com/cloudptio/octane
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/collector .


FROM alpine
RUN apk update && apk add --no-cache ca-certificates
COPY --from=builder /go/bin/collector /go/bin/collector
ENTRYPOINT ["/go/bin/collector"]
