#!/bin/bash
# Corporate pipeline runner with MITM proxy and custom CA support
# This script compiles and runs the corporate_main.go version

set -e

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check prerequisites
echo -e "${BLUE}🏢 Corporate CI/CD Pipeline Runner${NC}"
echo ""

# Verify credentials
if [ ! -f "../credentials/.env" ]; then
    echo -e "${YELLOW}⚠️  credentials/.env not found${NC}"
    echo "   Create it with:"
    echo "   cat > credentials/.env << EOF"
    echo "   CR_PAT=your_github_token"
    echo "   USERNAME=your_github_username"
    echo "   HTTP_PROXY=http://proxy.company.com:8080"
    echo "   HTTPS_PROXY=https://proxy.company.com:8080"
    echo "   EOF"
    exit 1
fi

# Load environment
set -a
source ../credentials/.env
set +a

# Check for CA certificates
if [ -d "credentials/certs" ]; then
    cert_count=$(find credentials/certs -name "*.pem" 2>/dev/null | wc -l)
    if [ "$cert_count" -gt 0 ]; then
        echo -e "${GREEN}✓ Found $cert_count CA certificate(s)${NC}"
    else
        echo -e "${YELLOW}⚠️  credentials/certs/ exists but no .pem files found${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No credentials/certs/ directory - corporate CA support disabled${NC}"
    echo "   Create with: mkdir -p credentials/certs"
    echo "   Then copy .pem files into it"
fi

# Check proxy configuration
if [ -n "$HTTP_PROXY" ] || [ -n "$HTTPS_PROXY" ]; then
    echo -e "${GREEN}✓ Proxy configured${NC}"
    [ -n "$HTTP_PROXY" ] && echo "   HTTP_PROXY=$HTTP_PROXY"
    [ -n "$HTTPS_PROXY" ] && echo "   HTTPS_PROXY=$HTTPS_PROXY"
else
    echo -e "${YELLOW}⚠️  No proxy configured (OK if not needed)${NC}"
fi

echo ""

# Compile corporate version
echo -e "${BLUE}Compiling corporate pipeline...${NC}"

# Build directly from corporate_main.go (it has its own main() function)
if go build -o railway-corporate-dagger-go corporate_main.go 2>&1; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${YELLOW}❌ Build failed${NC}"
    exit 1
fi

echo ""

# Run pipeline
echo -e "${BLUE}🚀 Executing corporate pipeline...${NC}"
echo ""

if [ "$DEBUG_CERTS" = "true" ]; then
    echo -e "${YELLOW}Debug mode enabled - will show certificate diagnostics${NC}"
    echo ""
fi

# Execute the binary
./railway-corporate-dagger-go

echo ""
echo -e "${GREEN}✅ Pipeline completed${NC}"
