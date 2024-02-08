#!/bin/bash
set -e
docker-compose --env-file ./spicedb/.env -f ./spicedb/docker-compose.yaml up -d