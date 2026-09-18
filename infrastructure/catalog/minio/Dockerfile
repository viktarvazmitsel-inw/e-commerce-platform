FROM golang:1.27-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git && \
    go install github.com/minio/minio@latest

FROM alpine:latest

COPY --from=builder /go/bin/minio /usr/bin/minio

EXPOSE 9000 9001

RUN mkdir -p /data

ENTRYPOINT ["minio", "server", "/data"]
