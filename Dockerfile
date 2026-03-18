ARG GO_VERSION=1.25.4
FROM golang:${GO_VERSION} AS builder

RUN mkdir /app
COPY . /app
WORKDIR /app

RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o main ./cmd

FROM alpine:3.23.3

ENV TZ=Asia/Bangkok

# update dependencies and fix CVE-2026-22184, CVE-2026-27171 (zlib >= 1.3.2-r0)
RUN apk update && \
    apk upgrade --no-cache && \
    apk add --no-cache bash shadow zlib>=1.3.2-r0 && \
    adduser -u 1000 -D nonroot && \
    rm -rf /var/cache/apk/*

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 3000
CMD ["/app/main"]