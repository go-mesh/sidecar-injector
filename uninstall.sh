#!/bin/bash

set -e
set -x

kubectl delete ServiceAccount mesher-sidecar-injector-service-account -n chassis
kubectl delete ClusterRole mesher-sidecar-injector-mesher-system
kubectl delete ClusterRoleBinding mesher-sidecar-injector-admin-role-binding-mesher-system
kubectl delete deployment sidecar-injector-webhook-mesher-deployment -n chassis
kubectl delete svc sidecar-injector-webhook-mesher-svc -n chassis
kubectl delete configmap mesher-configmap sidecar-injector-webhook-mesher-configmap -n chassis
kubectl delete MutatingWebhookConfiguration sidecar-injector-webhook-mesher-cfg
kubectl delete secrets sidecar-injector-webhook-mesher-certs -n chassis

kubectl delete ns chassis
