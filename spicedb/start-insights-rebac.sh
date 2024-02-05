#!/bin/bash
set -e
docker-compose --profile rebac -f ./spicedb/docker-compose.yaml up -d