FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o manifest ./cmd/manifest

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/manifest .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./manifest"]
