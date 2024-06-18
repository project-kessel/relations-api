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
echo "Deleting kind cluster -- \"kessel-relations-cluster\""

kind delete cluster --name kessel-relations-cluster
