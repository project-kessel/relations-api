#!/bin/bash
set -e
docker-compose --env-file ./spicedb/.env --profile rebac -f docker-compose.yaml down