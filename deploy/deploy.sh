#!/bin/bash

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
IMAGE=quay.io/ciam_authz/insights-rebac
IMAGE_TAG=latest

# Prepare bonfire env
VENV_DIR=~/bonfire_venv
mkdir -p $VENV_DIR
python3 -m venv $VENV_DIR
. $VENV_DIR/bin/activate

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

# Check if there is an existing NS
NAMESPACE=$(oc project -q)

if [[ -z "${NAMESPACE}" ]]; then
  echo "Namespace is not set"
    # Reserve a namespace
  bonfire namespace reserve --duration 8h
fi

NAMESPACE=$(oc project -q)
echo "Using Namespace:" $NAMESPACE

#Prepare the bonfire config yaml file
currentpath=$(pwd)
file_location=~/.config/bonfire/config.yaml
cat > $file_location <<EOF
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
  cat >> $file_location <<EOF
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
oc create configmap spicedb-schema --from-file=schema.yaml -n $NAMESPACE

#Deploy Relations service, spiceDB service and rbac service when $RBAC_ARGUMENT is not empty
bonfire deploy $RBAC_ARGUMENT relationships -n $NAMESPACE --local-config-method merge

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
  echo "Relations - Read(GET) - Sample CURL request"
  echo ""
  echo "curl -v -u ${USER}:${PASSWORD} '${BASE_URL}/api/authz/v1/relationships?filter.objectType=group&filter.objectId=bob_club&filter.relation=member'"
  echo ""
fi

echo "Relations - Write(POST) - Sample CURL request"
echo ""
echo "curl -v -u ${USER}:${PASSWORD} ${BASE_URL}/api/authz/v1/relationships -d '{ "touch": true, "relationships": [{"object": {"type": "group","id": "bob_club"},"relation": "member","subject": {"object": {"type": "user","id": "bob"}}}]}'"
