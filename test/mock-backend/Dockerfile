FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY main.go .

RUN go build -o mock-backend main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/mock-backend .

EXPOSE 9000

CMD ["./mock-backend"]