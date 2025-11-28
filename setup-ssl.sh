#!/bin/bash
# Install mkcert if not available
if ! command -v mkcert &> /dev/null; then
    echo "Installing mkcert..."
    if command -v yay &> /dev/null; then
        yay -S mkcert
    elif command -v pacman &> /dev/null; then
        sudo pacman -S mkcert
    else
        echo "Please install mkcert manually from: https://github.com/FiloSottile/mkcert"
        exit 1
    fi
fi

# Install local CA
echo "Installing local CA..."
mkcert -install

# Generate certificates
echo "Generating certificates..."
cd certs
mkcert localhost 127.0.0.1 192.168.8.27 ::1

echo "✓ Certificates created in certs/ directory"
echo "  - localhost+3.pem (certificate)"
echo "  - localhost+3-key.pem (private key)"
