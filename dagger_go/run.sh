#!/bin/bash
# Run the PRT Services Simulator Dagger Go CI/CD pipeline with GitHub registry authentication
# Usage: ./run.sh

set -e

# Load credentials from .env file if present
if [ -f "credentials/.env" ]; then
    set -a
    source credentials/.env
    set +a
fi

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

echo "🚀 Running PRT Services Simulator CI/CD Pipeline"
echo "   Repository: ${REPO_NAME:-prt_services_simulator}"
echo "   Image Name: ${IMAGE_NAME:-prt_services_simulator}"
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
