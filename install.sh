#!/bin/bash

set -x
set -e

namespace="servicecomb"
kubectl create -f deploy/namespace.yaml

bash -x deploy/signed-cert.sh --service sidecar-injector-webhook-mesher-svc \
 --secret sidecar-injector-webhook-mesher-certs --namespace ${namespace}

kubectl create -f deploy/mesherconfigmap.yaml -n ${namespace}
kubectl create -f deploy/configmap.yaml -n ${namespace}
kubectl create -f deploy/serviceaccount.yaml -n ${namespace}
kubectl create -f deploy/clusterrole.yaml -n ${namespace}
kubectl create -f deploy/clusterbinding.yaml -n ${namespace}
kubectl create -f deploy/deployment.yaml -n ${namespace}
kubectl create -f deploy/service.yaml -n ${namespace}
kubectl create -f deploy/mutatingwebhook.yaml -n ${namespace}
