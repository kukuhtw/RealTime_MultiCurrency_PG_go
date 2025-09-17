#!/bin/sh
set -e

# Wait for dependencies (optional)
if [ "$1" = "./api-gateway" ]; then
    echo "Waiting for gRPC services to be ready..."
    sleep 5
fi

# Run the binary
exec "$@"
