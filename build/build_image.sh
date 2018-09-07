#!/bin/bash

docker build -t gochassis/sidecar-injector:latest .
rm -rf sidecar-injector conf

docker push gochassis/sidecar-injector:latest