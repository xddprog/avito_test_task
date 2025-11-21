FROM golang:1.24.3-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o reviewer-service ./cmd/app

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/reviewer-service /app/reviewer-service
COPY api /app/api
COPY migrations /app/migrations

ENV HTTP_PORT=8080
EXPOSE 8080

CMD ["/app/reviewer-service"]
