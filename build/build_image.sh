#!/bin/bash

docker build -t gomesh/sidecar-injector:latest .
rm -rf sidecar-injector conf

docker push gomesh/sidecar-injector:latest
