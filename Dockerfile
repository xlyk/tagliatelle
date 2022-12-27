FROM golang:1.17 AS builder
ENV CGO_ENABLED=1 GOOS=linux
WORKDIR /go/src/tagliatelle
COPY . .
RUN go build -a -o /go/bin/tagliatelle ./cmd/tagliatelle/main.go

FROM alpine:latest
ENV CGO_ENABLED=1 GOOS=linux
RUN apk add --no-cache libc6-compat
COPY --from=builder /go/bin/tagliatelle /go/bin/tagliatelle
ENTRYPOINT ["/go/bin/tagliatelle"]
CMD ["-h"]
