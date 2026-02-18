#!/bin/bash
# Test script to validate that docker-compose fails when required environment variables are missing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing docker-compose environment variable validation...${NC}\n"

# Test 1: Missing GITHUB_WEBHOOK_SECRET
echo -e "${YELLOW}Test 1: Verifying compose fails without GITHUB_WEBHOOK_SECRET...${NC}"
unset GITHUB_WEBHOOK_SECRET
export API_AUTH_TOKEN="test-token"

if docker-compose config > /dev/null 2>&1; then
    echo -e "${RED}✗ FAILED: docker-compose should have failed without GITHUB_WEBHOOK_SECRET${NC}"
    exit 1
else
    echo -e "${GREEN}✓ PASSED: docker-compose correctly failed without GITHUB_WEBHOOK_SECRET${NC}\n"
fi

# Test 2: Missing API_AUTH_TOKEN
echo -e "${YELLOW}Test 2: Verifying compose fails without API_AUTH_TOKEN...${NC}"
export GITHUB_WEBHOOK_SECRET="test-secret"
unset API_AUTH_TOKEN

if docker-compose config > /dev/null 2>&1; then
    echo -e "${RED}✗ FAILED: docker-compose should have failed without API_AUTH_TOKEN${NC}"
    exit 1
else
    echo -e "${GREEN}✓ PASSED: docker-compose correctly failed without API_AUTH_TOKEN${NC}\n"
fi

# Test 3: Missing both required variables
echo -e "${YELLOW}Test 3: Verifying compose fails without both required variables...${NC}"
unset GITHUB_WEBHOOK_SECRET
unset API_AUTH_TOKEN

if docker-compose config > /dev/null 2>&1; then
    echo -e "${RED}✗ FAILED: docker-compose should have failed without required variables${NC}"
    exit 1
else
    echo -e "${GREEN}✓ PASSED: docker-compose correctly failed without required variables${NC}\n"
fi

# Test 4: With all required variables set
echo -e "${YELLOW}Test 4: Verifying compose succeeds with all required variables...${NC}"
export GITHUB_WEBHOOK_SECRET="test-secret"
export API_AUTH_TOKEN="test-token"

if docker-compose config > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASSED: docker-compose correctly succeeded with all required variables${NC}\n"
else
    echo -e "${RED}✗ FAILED: docker-compose should have succeeded with all required variables${NC}"
    exit 1
fi

echo -e "${GREEN}All validation tests passed!${NC}"
