#!/bin/bash

set -x
set -e
kubectl create -f deploy/namespace.yaml

bash -x deploy/signed-cert.sh --service sidecar-injector-webhook-mesher-svc --secret sidecar-injector-webhook-mesher-certs --namespace chassis

kubectl create -f deploy/mesherconfigmap.yaml -n chassis
kubectl create -f deploy/configmap.yaml -n chassis
kubectl create -f deploy/serviceaccount.yaml -n chassis
kubectl create -f deploy/clusterrole.yaml -n chassis
kubectl create -f deploy/clusterbinding.yaml -n chassis
kubectl create -f deploy/deployment.yaml -n chassis
kubectl create -f deploy/service.yaml -n chassis
kubectl create -f deploy/mutatingwebhook.yaml -n chassis
