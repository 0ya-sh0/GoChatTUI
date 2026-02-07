#!/usr/bin/env bash
set -e

APP_CLIENT=client
OUT=build

mkdir -p $OUT

echo "Building server (native)"
go build -o $OUT/server cmd/server/main.go

echo "Building testclient (native)"
go build -o $OUT/testclient cmd/testclient/main.go

echo "Building client for Linux amd64"
GOOS=linux GOARCH=amd64 \
go build -o $OUT/${APP_CLIENT}-linux-amd64 cmd/client/main.go

echo "Building client for macOS amd64"
GOOS=darwin GOARCH=amd64 \
go build -o $OUT/${APP_CLIENT}-darwin-amd64 cmd/client/main.go

echo "Building client for macOS arm64"
GOOS=darwin GOARCH=arm64 \
go build -o $OUT/${APP_CLIENT}-darwin-arm64 cmd/client/main.go

echo "Building client for Windows amd64"
GOOS=windows GOARCH=amd64 \
go build -o $OUT/${APP_CLIENT}-windows-amd64.exe cmd/client/main.go

echo "Done ✅"