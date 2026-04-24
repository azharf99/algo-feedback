# STAGE 1: Build
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o algonova-api ./cmd/api/main.go

# STAGE 2: Runtime (Menggunakan Alpine, hanya ~5MB base image)
FROM alpine:latest

# Instal TZData (untuk waktu) dan CA-Certificates saja
RUN apk add --no-cache tzdata ca-certificates

ENV TZ=Asia/Jakarta
WORKDIR /app

COPY --from=builder /app/algonova-api .

# Salin folder templates (sekarang isinya cuma gambar png/jpg untuk PDF)
COPY --from=builder /app/templates ./templates

RUN mkdir -p mediafiles

EXPOSE 8080
CMD ["./algonova-api"]