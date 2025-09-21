from golang:1-bookworm as builder
workdir /app
copy go.mod .
copy go.sum .
run go mod download
copy main.go .
copy backup ./backup
run go build -o /build/app

from debian:bookworm-slim
workdir /app
run apt-get update && apt-get install -y curl unzip && rm -rf /var/lib/apt/lists/*
run curl https://rclone.org/install.sh | bash
copy --from=builder /build/app .
entrypoint ["/app/app"]