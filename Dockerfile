FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o site-monitor ./cmd/main.go

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates libc6-compat

COPY --from=builder /app/site-monitor .
COPY --from=builder /app/docs/ ./docs/

RUN chmod +x ./site-monitor

CMD ["./site-monitor"]