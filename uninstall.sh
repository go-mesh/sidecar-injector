#!/bin/bash

set -e
set -x

namespace="servicecomb"

kubectl delete ServiceAccount mesher-sidecar-injector-service-account -n ${namespace}
kubectl delete ClusterRole mesher-sidecar-injector-mesher-system
kubectl delete ClusterRoleBinding mesher-sidecar-injector-admin-role-binding-mesher-system
kubectl delete deployment sidecar-injector-webhook-mesher-deployment -n ${namespace}
kubectl delete svc sidecar-injector-webhook-mesher-svc -n ${namespace}
kubectl delete configmap mesher-configmap sidecar-injector-webhook-mesher-configmap -n ${namespace}
kubectl delete MutatingWebhookConfiguration sidecar-injector-webhook-mesher-cfg
kubectl delete secrets sidecar-injector-webhook-mesher-certs -n ${namespace}

kubectl delete ns ${namespace}
