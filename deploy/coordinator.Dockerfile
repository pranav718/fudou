FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/coordinator ./cmd/coordinator

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /bin/coordinator /bin/coordinator

EXPOSE 8080

ENTRYPOINT ["/bin/coordinator"]
