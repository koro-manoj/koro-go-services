FROM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/worker ./cmd/worker

FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/api /usr/local/bin/api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]

FROM alpine:3.20 AS worker
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
