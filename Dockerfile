FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /api /app/api
COPY migrations ./migrations
COPY docs ./docs
EXPOSE 8080
CMD ["/app/api"]
