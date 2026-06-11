FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/shortener ./cmd/shortener

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/shortener /usr/local/bin/shortener
COPY web/public ./web/public

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/shortener"]

