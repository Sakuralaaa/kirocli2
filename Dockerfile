FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o kilocli2api .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/kilocli2api .
COPY --from=builder /app/web ./web

EXPOSE 4000

CMD ["./kilocli2api"]
