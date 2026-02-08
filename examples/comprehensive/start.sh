#!/bin/bash

# Start the comprehensive server with PostgreSQL
echo "🚀 Starting AuthSome Comprehensive Server..."
echo "📊 Database: postgres://postgres:***@localhost:5432/gameframework"
echo ""

# Export environment variables
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/gameframework?sslmode=disable"
export PORT="8081"
export DEBUG="true"

# Start the server
./comprehensive-server
