#!/bin/bash
# Build and test the Dagger Go CI/CD pipeline locally
# Usage: ./test.sh

set -e

echo "🧪 Testing PRT Services Simulator Dagger Go CI/CD Pipeline..."
echo ""

# Check for Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.22 or later."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Go version: $GO_VERSION"

# Download dependencies
echo ""
echo "📦 Downloading Go dependencies..."
go mod download

# Run unit tests
echo ""
echo "🧪 Running unit tests..."
go test -v -run Test

# Build the binary
echo ""
echo "🔨 Building PRT Services Simulator Dagger Go CLI..."
go build -o railway-dagger-go main.go

echo ""
echo "✅ Build successful!"
echo "   Binary: ./railway-dagger-go"
echo ""
echo "📖 To run the full pipeline with Docker:"
echo "   export CR_PAT=<your-github-token>"
echo "   export USERNAME=<your-github-username>"
echo "   ./railway-dagger-go"
