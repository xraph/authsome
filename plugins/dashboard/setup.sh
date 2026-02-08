#!/bin/bash

# Setup script for Dashboard Tailwind CSS

set -e

echo "🚀 Setting up Dashboard Tailwind CSS..."
echo ""

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    echo "❌ Error: npm is not installed"
    echo "Please install Node.js and npm from https://nodejs.org/"
    exit 1
fi

# Install dependencies
echo "📦 Installing dependencies..."
npm install

# Build CSS
echo ""
echo "🎨 Building CSS for the first time..."
npm run build:css

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Use 'make watch-css' or 'npm run watch:css' for development"
echo "  2. Use 'make build-css' or 'npm run build:css' for production"
echo "  3. Update layouts/layout.go to use output.css instead of CDN"
echo ""
