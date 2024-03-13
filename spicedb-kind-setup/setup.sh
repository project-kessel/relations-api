#!/bin/bash
set -oe errexit

if ! command -v kind &> /dev/null
then
    echo "kind could not be found"
    exit
fi

if ! command -v helm &> /dev/null
then
    echo "helm could not be found"
    exit
fi

if ! command -v kubectl &> /dev/null
then
    echo "kubectl could not be found"
    exit
fi

# desired cluster name; default is "kind"
echo "creating kind cluster"

kind create cluster --config ./spicedb-kind-setup/kind-kube/kind-ingress.config

echo "> waiting for kubernetes node(s) become ready"
kubectl wait --for=condition=ready node --all --timeout=60s

echo "deploy contour"
kubectl apply -f ./spicedb-kind-setup/kind-kube/contour.yaml


echo "Install spicedb-operator"
kubectl create namespace spicedb-operator
kubectl apply --server-side -f https://github.com/authzed/spicedb-operator/releases/latest/download/bundle.yaml -n spicedb-operator

echo "create spicedb namespace"
kubectl create namespace spicedb
echo "deploy postgres"
kubectl apply -f ./spicedb-kind-setup/postgres/secret.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/postgres/storage.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/postgres/postgresql.yaml -n spicedb

echo "deploy spicedb"
kubectl apply -f ./spicedb-kind-setup/spicedb-cr.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/svc-ingress.yaml -n spicedb

while [[ -z $(kubectl get deployments.apps -n spicedb spicedb-cr-spicedb -o jsonpath="{.status.readyReplicas}" 2>/dev/null) ]]; do
  echo "still waiting for spicedb"
  sleep 1
done
echo "spicedb is ready"
kubectl get ingresses.networking.k8s.io -n spicedb

echo "Deploying rebac service"
kubectl apply -f ./spicedb-kind-setup/rebac/secret.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/rebac/deployment.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/rebac/svc.yaml -n spicedb

while [[ -z $(kubectl get deployments.apps -n spicedb relationships -o jsonpath="{.status.readyReplicas}" 2>/dev/null) ]]; do
  echo "still waiting for relationships"
  sleep 1
done

echo "Route"
kubectl get ingresses.networking.k8s.io -n spicedb

echo "Sample curl to relationships api"
echo ""
echo  "curl http://relationships.127.0.0.1.nip.io/api/authz/v1/relationships -d '{ "touch": true, "relationships": [{"object": {"type": "group","id": "bob_club"},"relation": "member","subject": {"object": {"type": "user","id": "bob"}}}]}'"