FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/node ./cmd/node

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /bin/node /bin/node

EXPOSE 9001

ENTRYPOINT ["/bin/node"]
