.PHONY: help install build-css watch-css clean

# Default target
help:
	@echo "Available targets:"
	@echo "  install    - Install npm dependencies"
	@echo "  build-css  - Build production CSS"
	@echo "  watch-css  - Watch and rebuild CSS on changes"
	@echo "  clean      - Remove generated files"

# Install dependencies
install:
	@echo "📦 Installing dependencies..."
	@npm install

# Build CSS for production (minified)
build-css:
	@echo "🎨 Building CSS for production..."
	@npm run build:css
	@echo "✅ CSS built to static/css/output.css"

# Watch CSS for development
watch-css:
	@echo "👀 Watching CSS files for changes..."
	@npm run watch:css

# Clean generated files
clean:
	@echo "🧹 Cleaning generated files..."
	@rm -f static/css/output.css
	@rm -rf node_modules
	@echo "✅ Clean complete"
