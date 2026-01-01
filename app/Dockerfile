FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY main.go .

RUN go build -o app main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /build/app .

EXPOSE 5000

CMD ["./app"]
