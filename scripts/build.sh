#!/bin/bash

# Build script for GO-BOOTP server

echo "Building GO-BOOTP server..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Change to project directory
cd "$PROJECT_DIR"

# Build the binary
go build -o bootpd cmd/bootpd/main.go

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: bootpd"
else
    echo "Build failed!"
    exit 1
fi