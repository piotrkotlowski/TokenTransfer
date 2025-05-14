# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder


RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download


COPY . .
RUN go build -o server .


FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server ./
EXPOSE 8080
CMD ["./server"]