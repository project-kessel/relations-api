#!/bin/bash
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

# Reserve a namespace
bonfire namespace reserve --duration 8h
NAMESPACE=$(oc config view --minify -o 'jsonpath={..namespace}')

if [[ -z "${NAMESPACE}" ]]; then
  echo "Namespace is not set"
  exit 1
fi
echo "Using Namespace:" $NAMESPACE

#Prepare the bonfire config yaml file
currentpath=$(pwd)
file_location=~/.config/bonfire/config.yaml
cat > $file_location <<EOF
apps:
- name: relationships
  components:
    - name:  relationships
      host: local
      repo: $currentpath
      path: clowdapp.yaml
      parameters:
        NAMESPACE: $NAMESPACE
        IMAGE: $IMAGE
        IMAGE_TAG: $IMAGE_TAG
EOF

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

#Deploy Relations service, spiceDB service
bonfire deploy relationships -n $NAMESPACE --local-config-method override 
