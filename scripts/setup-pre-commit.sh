#!/bin/bash

# Install pre-commit if not already installed
if ! command -v pre-commit &> /dev/null; then
    echo "Installing pre-commit..."
    pip install pre-commit
fi

# Install golangci-lint if not already installed
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
fi

# Install pre-commit hooks
echo "Installing pre-commit hooks..."
pre-commit install

echo "Setup complete! Pre-commit hooks are now installed."
echo "They will run tests and linting before each commit." 