#!/bin/bash
set -e

# Script to regenerate Go client SDK
echo "🔧 Regenerating Go client SDK..."

cd "$(dirname "$0")/.."

# Build the generator
echo "📦 Building generator..."
go build -o /tmp/authsome-gen ./cmd/authsome-cli/

# Generate the client
echo "🚀 Generating Go client..."
/tmp/authsome-gen generate client --lang go --output ./clients

# Check for common errors
echo "🔍 Checking for generation errors..."
if grep -r "authsome\. \`json" clients/go/plugins/ 2>/dev/null; then
    echo "❌ Found empty type names (authsome. )"
    exit 1
fi

if grep -r "\*\*redis" clients/go/ 2>/dev/null; then
    echo "❌ Found double pointers (**redis)"
    exit 1
fi

# Count redeclarations
echo "🔍 Checking for type redeclarations..."
duplicates=$(grep -h "^type SignInRequest\|^type SignInResponse\|^type SignUpResponse" clients/go/*.go clients/go/plugins/*/*.go 2>/dev/null | sort | uniq -d | wc -l)
if [ "$duplicates" -gt 0 ]; then
    echo "❌ Found type redeclarations"
    grep -h "^type SignInRequest\|^type SignInResponse\|^type SignUpResponse" clients/go/*.go clients/go/plugins/*/*.go 2>/dev/null | sort | uniq -c
    exit 1
fi

echo "✅ Go client generated successfully!"
echo "📁 Output: ./clients/go/"

