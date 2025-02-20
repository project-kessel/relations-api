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

echo "Deploying relations-api service"
kubectl apply -f ./spicedb-kind-setup/relations-api/secret.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/relations-api/deployment.yaml -n spicedb
kubectl apply -f ./spicedb-kind-setup/relations-api/svc.yaml -n spicedb

while [[ -z $(kubectl get deployments.apps -n spicedb relationships -o jsonpath="{.status.readyReplicas}" 2>/dev/null) ]]; do
  echo "still waiting for relationships"
  sleep 1
done

echo "Route"
kubectl get ingresses.networking.k8s.io -n spicedb

echo "Relations - Write(POST) - Sample CURL request"
echo ""
JSON_DATA='{"tuples":[{"resource":{"id":"bob_club","type":{"name":"group","namespace":"rbac"}},"relation":"member","subject":{"subject":{"id":"bob","type":{"name":"principal","namespace":"rbac"}}}}]}'
echo "curl -v http://relationships.127.0.0.1.nip.io:8000/api/authz/v1beta1/tuples -H 'Content-Type: application/json' -d '$JSON_DATA'"
