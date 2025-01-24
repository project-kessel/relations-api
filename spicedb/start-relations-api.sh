#!/bin/bash
set -e
# Function to check if a command is available
source ./spicedb/check_docker_podman.sh
NETWORK_CHECK=$(${DOCKER} network ls --filter name=kessel --format json)
if [[ -z "${NETWORK_CHECK}" || "${NETWORK_CHECK}" == "[]" ]]; then ${DOCKER} network create kessel; fi
echo "Downloading latest schema"
curl -o deploy/schema.zed https://raw.githubusercontent.com/RedHatInsights/rbac-config/refs/heads/master/configs/prod/schemas/schema.zed
${DOCKER} compose --env-file ./spicedb/.env --profile relations-api -f ./docker-compose.yaml up -d --build
