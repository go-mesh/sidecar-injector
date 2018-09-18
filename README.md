# sidecar-injector  
[![Build Status](https://travis-ci.org/go-mesh/sidecar-injector.svg?branch=master)](https://travis-ci.org/go-mesh/sidecar-injector)

## Need to update the license file

## Prerequisites
```
1. Ensure that the kubernetes cluster has at least 1.9 or above.
2. Ensure that MutatingAdmissionWebhook controllers are enabled
3. Ensure that the admissionregistration.k8s.io/v1beta1 API is enabled
```
Verification:
```
kubectl api-versions | grep admissionregistration.k8s.io/v1beta1
```
The output should be:
```
admissionregistration.k8s.io/v1beta1
```

OR

```
ps -ef | grep kube-apiserver | grep enable-admission-plugins
```
Output should be:
```
--enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota
```

Follow [admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#how-do-i-turn-on-an-admission-controller)
If enable-admission-plugins has not been enabled or it doesnot contain the MutatingAdmissionWebhook and ValidatingAdmissionWebhook 

## Quick Start

```
bash -x install.sh

kubectl label namespace chassis sidecar-injector=enabled
[root@mstnode ~]# kubectl get namespace -L sidecar-injector
NAME          STATUS    AGE       SIDECAR-INJECTOR
default       Active    18h
kube-public   Active    18h
kube-system   Active    18h
chassis       Active    3m        enabled

## Verify

cd example/sc

kubectl create -f client.yaml -n chassis
kubectl create -f server.yaml -n chassis

kubectl get pods -n chassis

NAME                   READY     STATUS    RESTARTS   AGE
client-mesher          2/2       Running   0          33s
server-mesher          2/2       Running   0          12s

```

## Build

1. Setup dependency

   The repo use glide as the dependency management tool for its Go codebase. 
To Install `glide` follow [glide](https://github.com/Masterminds/glide)

2. Build binary, image and push to docker hub

```
1. clone sidecar-injector code 

2. setup a GOPATH

3. cd sidecar-injector 

4. bash -x build.sh
```

## Install

```
bash -x install.sh
```

## Enable sidecar-injector for namespace on which sidecar pod has been deployed

1. The sidecar injector webhook should be running
```
[root@mstnode ~]# kubectl get pods -n chassis
NAME                                                          READY     STATUS    RESTARTS   AGE
sidecar-injector-webhook-mesher-deployment-8576646db8-x6f56   1/1       Running   0          20s

[root@mstnode ~]# kubectl get deployment -n chassis
NAME                                         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
sidecar-injector-webhook-mesher-deployment   1         1         1            1           1m
```

2. Label the chassis namespace with `sidecar-injector=enabled`
```
kubectl label namespace chassis sidecar-injector=enabled
[root@mstnode ~]# kubectl get namespace -L sidecar-injector
NAME          STATUS    AGE       SIDECAR-INJECTOR
default       Active    18h
kube-public   Active    18h
kube-system   Active    18h
chassis      Active    3m        enabled
```

## Deploy example 

1. Deploy an app in Kubernetes cluster, take `client` app as an example

```
[root@mstnode ~]# cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Pod
metadata:
  name: client
  namespace: chassis
  annotations:
    sidecar.mesher.io/inject: "yes"
  labels:
    app: client
    version: 0.0.1
spec:
  containers:
    - name: client
      image: xiaoliang/client-go
      env:
        - name: TARGET
          value: http://server-mesher/
        - name: http_proxy
          value: http://127.0.0.1:30101/
      ports:
        - containerPort: 9000
EOF
```

Notes: The injector will use label `app` value as the microservice name,
if the metadata does not set the label `app`, injector will use the `Pod` name by default.

## Verification

1. Verify sidecar container injected
```
[root@mstnode ~]# kubectl get pods -n chassis
NAME            READY     STATUS    RESTARTS   AGE
client          2/2       Running   0          12s
```

## Clean
```
bash -x uninstall.sh
```
