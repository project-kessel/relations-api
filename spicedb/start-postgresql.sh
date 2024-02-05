set -e
# Function to check if a command is available
source ./spicedb/check_docker_podman.sh

$DOCKER network create spicedb-net || true

$DOCKER run \
  --name spicedb-datastore \
  --net spicedb-net \
  --restart=always \
  -e POSTGRES_PASSWORD=$(cat ./spicedb/secrets/db.password) \
  -e POSTGRES_USER=$(cat ./spicedb/secrets/db.user) \
  -e POSTGRES_DB=$(cat ./spicedb/secrets/db.name) \
  -p $(cat  ./spicedb/secrets/secrets/db.port):5432 \
  -d postgres:latest