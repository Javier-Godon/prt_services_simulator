#!/bin/bash
# Run the Railway Dagger Go CI/CD pipeline with GitHub registry authentication
# Usage: ./run.sh

set -e

# Check required environment variables
if [ -z "$CR_PAT" ]; then
    echo "❌ CR_PAT environment variable is not set"
    echo "   Set it to your GitHub Personal Access Token with 'write:packages' scope"
    exit 1
fi

if [ -z "$USERNAME" ]; then
    echo "❌ USERNAME environment variable is not set"
    echo "   Set it to your GitHub username"
    exit 1
fi

echo "🚀 Running Railway Framework CI/CD Pipeline"
echo "   Repository: ${REPO_NAME:-railway_oriented_java}"
echo "   Image Name: ${IMAGE_NAME:-railway_oriented_java}"
echo "   GitHub User: $USERNAME"
echo ""

# Check if binary exists, build if not
if [ ! -f ./railway-dagger-go ]; then
    echo "📦 Building railway-dagger-go binary..."
    go mod download dagger.io/dagger
    go mod tidy
    go build -o railway-dagger-go main.go
fi

# Run the pipeline binary
./railway-dagger-go
