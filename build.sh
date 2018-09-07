#!/bin/bash
set -e
set -x

glide install
rm -rf vendor/k8s.io/apiextensions-apiserver/vendor/k8s.io/apiserver/pkg/util/feature/
rm -rf vendor/k8s.io/kubernetes/vendor/k8s.io

appname="sidecar-injector"

rm -rf $appname

BUILD_PATH=$(cd $(dirname $0);pwd)

echo $BUILD_PATH
cd $BUILD_PATH

CGO_ENABLED=0 GO_EXTLINK_ENABLED=0 go build --ldflags '-s -w -extldflags "-static"' -a -o $appname

cp -r conf/ $appname build/; cd $BUILD_PATH/build

bash -x build_image.sh
