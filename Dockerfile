# build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# install build dependencies
RUN apk add --no-cache git

# copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# copy source code
COPY . .

# build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/api

# final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# copy the binary from builder
COPY --from=builder /app/api-gateway .

# expose port
EXPOSE 8080

# run the binary
CMD ["./api-gateway"]
