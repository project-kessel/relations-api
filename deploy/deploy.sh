#!/bin/bash
set -e

# Define the minimum required version
MIN_BONFIRE_VERSION="5.8.0"

RBAC_ARGUMENT="$1"
RBAC_DIR="$2"
if [ "$RBAC_ARGUMENT" == "rbac" ]; then
 if [ ! -d "$RBAC_DIR" ]; then
    echo "The directory $RBAC_DIR does not exist."
    echo "Please specify local directory(absolute path) to copy of https://github.com/RedHatInsights/insights-rbac repository."
    exit
  fi
fi

source ../.secrets/postgres.env

# Export tags
IMAGE=quay.io/cloudservices/kessel-relations
IMAGE_TAG=latest

# Function to check if a command is available
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# pre-flight checks
if command_exists bonfire; then
  echo "Bonfire is OK "
else
  echo "bonfire needs to be installed"
  exit 1
fi

CURRENT_BONFIRE_VERSION=$(bonfire version | cut -d' ' -f3)

if [[ $(printf '%s\n' "$MIN_BONFIRE_VERSION" "$CURRENT_BONFIRE_VERSION" | sort -V | head -n1) != "$MIN_BONFIRE_VERSION" ]]; then
  echo "Current bonfire version ($CURRENT_BONFIRE_VERSION) is less than required version ($MIN_BONFIRE_VERSION)."
  echo "Please upgrade bonfire with command:"
  echo "pip install --upgrade crc-bonfire"
  exit 1
fi

# Check if there is an existing NS
RESERVATIONS=$(bonfire namespace list -m)

if [ "${RESERVATIONS}" == "no namespaces found" ]; then
  echo "Namespace is not set"
    # Reserve a namespace
  NAMESPACE=$(bonfire namespace reserve --duration 8h)
fi

if [[ -z "${NAMEPACE}" ]]; then
  NAMESPACE=$(oc project -q)
fi

echo "Using Namespace:" $NAMESPACE

#Prepare the bonfire config yaml file
currentpath=$(pwd)
config_file_location=./local_bonfire_config.yaml
cat > $config_file_location <<EOF
apps:
- name: relationships
  components:
    - name: relationships
      host: local
      repo: $currentpath
      path: clowdapp.yaml
      parameters:
        NAMESPACE: $NAMESPACE
        IMAGE: $IMAGE
        IMAGE_TAG: $IMAGE_TAG
EOF

if [[ "$RBAC_ARGUMENT" == "rbac" ]]; then
  cat >> $config_file_location <<EOF
- name: rbac
  components:
    - name: rbac
      host: local
      repo: $RBAC_DIR/deploy
      path: rbac-clowdapp.yml
      parameters:
        IMAGE: quay.io/lpichler/insights-rbac
        IMAGE_TAG: rebac
EOF
fi

# Create postgres pod,service and the spiceDB secret
oc process -f postgres.yaml -p NAMESPACE=$NAMESPACE -p POSTGRES_USER=$POSTGRES_USER -p POSTGRES_PASSWORD=$POSTGRES_PASSWORD -p POSTGRES_DB=$POSTGRES_DB | oc apply --wait=true -f -

# check the postgres service and secret are created
while [[ -z $(oc get deployments.apps -n $NAMESPACE postgres -o jsonpath="{.status.readyReplicas}" 2>/dev/null) ]]; do
  echo "still waiting for postgres"
  sleep 1
done
echo "postgress is ready"

# Create spiceDB bootstrap schema configmap

if ! oc get configmap spicedb-schema >/dev/null 2>&1; then
    oc create configmap spicedb-schema --from-file=schema.yaml -n $NAMESPACE
else
    echo "Configmap 'spicedb-schema' already exists."
fi

#Deploy Relations service, spiceDB service and rbac service when $RBAC_ARGUMENT is not empty
bonfire deploy $RBAC_ARGUMENT relationships -n $NAMESPACE --local-config-method merge --local-config-path $config_file_location

ROUTE=$(oc get routes --selector='app=relationships' -o jsonpath='{.items[*].spec.host}')
BASE_URL="https://$ROUTE"

echo ""
echo "Route: ${BASE_URL}/api/authz/v1/relationships"

USER="$(oc get secrets env-$NAMESPACE-keycloak --template={{.data.defaultUsername}} | base64 -d)"
PASSWORD="$( oc get secrets env-$NAMESPACE-keycloak --template={{.data.defaultPassword}} | base64 -d)"

echo ""
echo "user: ${USER}"
echo "pass: ${PASSWORD}"
echo ""

if [[ "$RBAC_ARGUMENT" == "rbac" ]]; then
  echo "RBAC - status request consist creation of relations(image from PR https://github.com/RedHatInsights/insights-rbac/pull/1060)"
  echo ""
  echo "curl -v -u ${USER}:${PASSWORD} ${BASE_URL}/api/rbac/v1/status/"
  echo ""
  # Revisit after the HTTP streaming decision
  # echo "Relations - Read(GET) - Sample CURL request"
  # echo ""
  # echo "curl -v -u ${USER}:${PASSWORD} '${BASE_URL}/api/authz/v1/relationships?filter.objectType=group&filter.objectId=bob_club&filter.relation=member'"
  # echo ""
fi

echo "Relations - Write(POST) - Sample CURL request"
echo ""
JSON_DATA='{ "tuples": [{"resource": {"type": {"type": "group"},"id": "bob_club2"},"relation": "member","subject": {"subject": {"type": {"type": "user"},"id": "bob2"}}}]}'
echo "curl -v -u ${USER}:${PASSWORD} ${BASE_URL}/api/authz/v1beta1/tuples -H 'Content-Type: application/json' -d '$JSON_DATA'"
