FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY . ./

# Ensure go.sum is generated and all dependencies are fetched inside the container.
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gcr-api ./cmd/gcr-api

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/gcr-api /usr/local/bin/gcr-api

EXPOSE 8080

ENTRYPOINT ["gcr-api"]


