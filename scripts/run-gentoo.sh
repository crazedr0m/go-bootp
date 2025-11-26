#!/bin/bash

# Run script for GO-BOOTP server on Gentoo Linux

echo "Starting GO-BOOTP server on Gentoo Linux..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Change to project directory
cd "$PROJECT_DIR"

# Check if binary exists
if [ ! -f "bootpd" ]; then
    echo "bootpd binary not found. Building..."
    go build -o bootpd cmd/bootpd/main.go
fi

# Run with sudo to bind to privileged port 67
echo "Running GO-BOOTP server on Gentoo (requires root privileges)..."
sudo ./bootpd -config configs/dhcpd.conf