#!/bin/bash
# Development environment setup script

export WCLOUD_ENDPOINT="https://api-dev.waveterm.dev/central"
export WCLOUD_WS_ENDPOINT="wss://wsapi-dev.waveterm.dev/"

echo "Environment variables set for development"
echo "WCLOUD_ENDPOINT: $WCLOUD_ENDPOINT"
echo "WCLOUD_WS_ENDPOINT: $WCLOUD_WS_ENDPOINT"

# Run yarn dev with the environment variables
yarn dev