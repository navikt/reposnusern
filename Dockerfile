# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Kopier mod-filer og last ned deps først for cache
COPY go.mod go.sum ./
RUN go mod download

# Kopier resten av koden
COPY . .

# Bygg binæren
RUN go build -o reposnusern ./cmd/reposnusern

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/reposnusern /reposnusern

CMD ["/reposnusern"]
