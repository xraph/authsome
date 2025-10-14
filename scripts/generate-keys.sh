#!/bin/bash

# Script to generate RSA key pairs for OIDC Provider testing
# This creates proper RSA keys that can be loaded from files

set -e

KEYS_DIR="./keys"
PRIVATE_KEY="$KEYS_DIR/oidc-private.pem"
PUBLIC_KEY="$KEYS_DIR/oidc-public.pem"

echo "Generating RSA key pairs for OIDC Provider..."

# Create keys directory if it doesn't exist
mkdir -p "$KEYS_DIR"

# Generate private key (2048-bit RSA)
echo "Generating private key..."
openssl genrsa -out "$PRIVATE_KEY" 2048

# Extract public key from private key
echo "Extracting public key..."
openssl rsa -in "$PRIVATE_KEY" -pubout -out "$PUBLIC_KEY"

# Set appropriate permissions
chmod 600 "$PRIVATE_KEY"
chmod 644 "$PUBLIC_KEY"

echo "Keys generated successfully:"
echo "  Private key: $PRIVATE_KEY"
echo "  Public key:  $PUBLIC_KEY"
echo ""
echo "You can now configure the OIDC Provider to use these keys:"
echo ""
echo "auth:"
echo "  oidcprovider:"
echo "    keys:"
echo "      privateKeyPath: \"$PRIVATE_KEY\""
echo "      publicKeyPath: \"$PUBLIC_KEY\""
echo ""
echo "Note: In production, store private keys securely and never commit them to version control!"