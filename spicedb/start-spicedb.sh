#!/bin/bash
set -e
docker-compose --env-file ./spicedb/.env -f ./docker-compose.yaml up -d