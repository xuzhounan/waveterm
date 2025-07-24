#!/bin/bash

# Copyright 2025, Command Line Inc.
# SPDX-License-Identifier: Apache-2.0

# Test script for Widget API workspace name functionality
# This script tests the new GET /api/v1/widgets/workspace/name/{workspace_name} endpoint

set -e

# Configuration
BASE_URL="http://localhost:8090"
API_ENDPOINT="/api/v1/widgets"

echo "🧪 Testing Widget API - Get Workspace by Name"
echo "============================================="

# Test 1: List all workspaces first to see available workspace names
echo
echo "📋 Test 1: List all workspaces"
echo "GET ${BASE_URL}${API_ENDPOINT}/workspaces"
curl -s -X GET "${BASE_URL}${API_ENDPOINT}/workspaces" | jq '.'

# Test 2: Try to get a workspace by name (using a common name like "Default" or the first workspace)
echo
echo "🔍 Test 2: Get workspace by name - 'Default'"
echo "GET ${BASE_URL}${API_ENDPOINT}/workspace/name/Default"
response=$(curl -s -X GET "${BASE_URL}${API_ENDPOINT}/workspace/name/Default")
echo $response | jq '.'

# Check if we got a valid response
if echo $response | jq -r '.success' | grep -q true; then
    echo "✅ Successfully found workspace 'Default'"
    workspace_id=$(echo $response | jq -r '.workspace.workspace_id')
    echo "   Workspace ID: $workspace_id"
else
    echo "ℹ️  Workspace 'Default' not found, this is expected if no workspace has this name"
fi

# Test 3: Try to get a workspace by a name that likely doesn't exist
echo
echo "🚫 Test 3: Get workspace by non-existent name - 'NonExistentWorkspace'"
echo "GET ${BASE_URL}${API_ENDPOINT}/workspace/name/NonExistentWorkspace"
response=$(curl -s -X GET "${BASE_URL}${API_ENDPOINT}/workspace/name/NonExistentWorkspace")
echo $response | jq '.'

# Check if we got appropriate error response
if echo $response | jq -r '.success' | grep -q false; then
    echo "✅ Correctly returned error for non-existent workspace"
else
    echo "❌ Expected error response for non-existent workspace"
fi

# Test 4: Try to get a workspace with empty name (should return 400)
echo
echo "❓ Test 4: Get workspace with empty name"
echo "GET ${BASE_URL}${API_ENDPOINT}/workspace/name/"
response=$(curl -s -X GET "${BASE_URL}${API_ENDPOINT}/workspace/name/")
echo $response | jq '.'

# Test 5: Case insensitive test (create a workspace with a known name first, then test)
echo
echo "🔤 Test 5: Case insensitive name matching"
echo "This test assumes there's a workspace name available to test with"
echo "If you have workspaces, try accessing them with different cases"

# Test 6: URL encoding test (workspace names with spaces)
echo
echo "🌐 Test 6: URL encoded workspace name (spaces)"
echo "GET ${BASE_URL}${API_ENDPOINT}/workspace/name/My%20Workspace"
response=$(curl -s -X GET "${BASE_URL}${API_ENDPOINT}/workspace/name/My%20Workspace")
echo $response | jq '.'

echo
echo "🎯 All tests completed!"
echo
echo "Expected behavior:"
echo "- Test 1: Should list all available workspaces"
echo "- Test 2: Should find workspace if 'Default' exists, or return not found"
echo "- Test 3: Should return 404 with error message for non-existent workspace"
echo "- Test 4: Should return 400 for empty workspace name"
echo "- Test 5: Should work with case-insensitive matching"
echo "- Test 6: Should handle URL-encoded names properly"
echo
echo "🔧 To run this test, ensure Wave Terminal server is running on localhost:8090"
echo "   You can start it by running: go run cmd/server/main-server.go"