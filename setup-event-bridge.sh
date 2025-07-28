#!/bin/bash

# Setup Event Bridge between MCP Server and Development Environment
# This script configures cross-server event synchronization

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🌉 Setting up Event Bridge for Wave Terminal${NC}"

# Check if development server is running
DEV_PORT=""
if lsof -i :50760 >/dev/null 2>&1; then
    DEV_PORT="50760"
    echo -e "${GREEN}✅ Development server detected on port 50760${NC}"
elif lsof -i :50761 >/dev/null 2>&1; then
    DEV_PORT="50761"  
    echo -e "${GREEN}✅ Development server detected on port 50761${NC}"
else
    echo -e "${YELLOW}⚠️  No development server detected. Start with 'task dev' first.${NC}"
    exit 1
fi

# Check if MCP server is running
MCP_PORT=""
if lsof -i :60289 >/dev/null 2>&1; then
    MCP_PORT="60289"
    echo -e "${GREEN}✅ MCP server detected on port 60289${NC}"
else
    echo -e "${YELLOW}⚠️  No MCP server running. Start with './persistent-server.sh start'${NC}"
    exit 1
fi

DEV_BASE_URL="http://localhost:${DEV_PORT}"
MCP_BASE_URL="http://localhost:${MCP_PORT}"

echo -e "${BLUE}🔧 Configuring event bridge...${NC}"

# Enable bridge on development server
echo "Enabling bridge on development server..."
curl -s -X POST "${DEV_BASE_URL}/api/v1/bridge/config" \
    -H "Content-Type: application/json" \
    -d '{"action": "enable"}' >/dev/null || {
    echo -e "${RED}❌ Failed to enable bridge on development server${NC}"
    exit 1
}

# Add MCP server as remote target for development server
echo "Adding MCP server as remote target..."
curl -s -X POST "${DEV_BASE_URL}/api/v1/bridge/config" \
    -H "Content-Type: application/json" \
    -d "{\"action\": \"add_remote\", \"url\": \"${MCP_BASE_URL}\"}" >/dev/null || {
    echo -e "${RED}❌ Failed to add MCP server as remote${NC}"
    exit 1
}

# Enable bridge on MCP server  
echo "Enabling bridge on MCP server..."
curl -s -X POST "${MCP_BASE_URL}/api/v1/bridge/config" \
    -H "Content-Type: application/json" \
    -d '{"action": "enable"}' >/dev/null || {
    echo -e "${RED}❌ Failed to enable bridge on MCP server${NC}"
    exit 1
}

# Add development server as remote target for MCP server
echo "Adding development server as remote target..."
curl -s -X POST "${MCP_BASE_URL}/api/v1/bridge/config" \
    -H "Content-Type: application/json" \
    -d "{\"action\": \"add_remote\", \"url\": \"${DEV_BASE_URL}\"}" >/dev/null || {
    echo -e "${RED}❌ Failed to add development server as remote${NC}"
    exit 1
}

echo -e "${GREEN}✅ Event bridge configured successfully!${NC}"
echo
echo -e "${BLUE}📊 Bridge Status:${NC}"

# Show bridge status for both servers
echo "Development Server Bridge Status:"
curl -s "${DEV_BASE_URL}/api/v1/bridge/status" | jq . 2>/dev/null || echo "  (status unavailable)"

echo
echo "MCP Server Bridge Status:"  
curl -s "${MCP_BASE_URL}/api/v1/bridge/status" | jq . 2>/dev/null || echo "  (status unavailable)"

echo
echo -e "${GREEN}🎉 Event bridge is now active!${NC}"
echo -e "${YELLOW}💡 Widgets created via MCP API will now appear in real-time in Wave Terminal${NC}"