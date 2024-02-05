#!/bin/bash
# Function to check if a command is available
source ./spicedb/check_docker_podman.sh

set -e

$DOCKER stop spicedb-datastore

$DOCKER rm spicedb-datastore

$DOCKER network rm spicedb-net

docker-compose -f ./spicedb/docker-compose.yaml down


