#!/bin/bash

# Fix environment for Wave Terminal development

# Enable corepack
corepack enable

# Make sure homebrew is in PATH
export PATH="/opt/homebrew/bin:$PATH"

# Prepare yarn 4.6.0
corepack prepare yarn@4.6.0 --activate

echo "Environment fixed for development"
echo "Go version: $(go version)"
echo "Yarn version: $(yarn --version)"
echo "Task version: $(task --version)"