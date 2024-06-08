#!/bin/sh

until cockroach sql --insecure --execute="SELECT 1;" &> /dev/null; do
  echo "Waiting for CockroachDB to be ready..."
  sleep 2
done

# Create the database
cockroach sql --insecure --execute="CREATE DATABASE IF NOT EXISTS linkgraph;"

echo "Initialization complete!"