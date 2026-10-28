# --- Byggfase ---
FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app

# Kopier mod-filer og last ned deps først for cache
COPY go.mod go.sum ./
RUN go mod download

# Kopier resten av koden
COPY . .

# Bygg binæren
RUN CGO_ENABLED=0 go build -o reposnusern ./cmd/reposnusern

# --- Runtime stage ---
FROM cgr.dev/chainguard/static:latest

COPY --from=builder /app/reposnusern /reposnusern

CMD ["/reposnusern"]
