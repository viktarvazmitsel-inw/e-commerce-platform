FROM golang:1.27-alpine

ARG WWWUSER=1000
ARG WWWGROUP=1000

RUN apk add --no-cache shadow \
    && groupmod -g ${WWWGROUP} node 2>/dev/null || groupadd -g ${WWWGROUP} appgroup 2>/dev/null || true \
    && usermod -u ${WWWUSER} node 2>/dev/null || useradd -u ${WWWUSER} -g ${WWWGROUP} appuser 2>/dev/null || true

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .

EXPOSE 8080

CMD ["go", "run", "./cmd/api/main.go"]
