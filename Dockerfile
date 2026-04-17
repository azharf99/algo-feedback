# ==========================================
# STAGE 1: Build file biner Go dengan Go 1.26
# ==========================================
FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Menyalin file modul terlebih dahulu (untuk caching)
COPY go.mod go.sum ./
RUN go mod download

# Menyalin seluruh kode sumber
COPY . .

# Membangun aplikasi
RUN CGO_ENABLED=0 GOOS=linux go build -o algonova-api ./cmd/api/main.go

# ==========================================
# STAGE 2: Runtime Minimalis dengan Chrome (Debian 12 Bookworm)
# ==========================================
FROM debian:bookworm-slim

# SANGAT PENTING: Instal Chromium dan Fonts
RUN apt-get update && apt-get install -y \
    chromium \
    fonts-liberation \
    fonts-dejavu \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Atur zona waktu ke WIB (Asia/Jakarta)
ENV TZ=Asia/Jakarta

WORKDIR /app

# Salin biner dari Stage 1
COPY --from=builder /app/algonova-api .

# Salin folder templates agar HTML bisa dibaca
COPY --from=builder /app/templates ./templates

# Buat folder untuk menampung PDF
RUN mkdir -p mediafiles

EXPOSE 8080

# Jalankan aplikasi
CMD ["./algonova-api"]