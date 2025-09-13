FROM golang:1.25.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o lazyos .

FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache upgrade

COPY --from=builder /app/lazyos .

CMD ["./lazyos"]