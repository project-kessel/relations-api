#!/bin/bash
set -e
# Function to check if a command is available
source ./spicedb/check_docker_podman.sh
${DOCKER} compose --env-file ./spicedb/.env --profile relations-api -f docker-compose.yaml down