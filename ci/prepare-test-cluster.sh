#!/bin/bash -xe

# This script should be called on any new cluster used by the e2e test
# It installs the things needed for glooshot tests to run
# These include:
#  - Supergloo
#  - Istio
#  - Prometheus
#  - The modified bookinfo app


echo Preparing cluster for glooshot e2e tests.
echo Note: this script will fail if not run from root dir.
# simple heuristic dir check to fail fast if not run from correct dir
ls -la ci/prepare-test-cluster.sh

supergloo init

sleep 60

supergloo install istio --name istio --installation-namespace istio-system --mtls=true --auto-inject=true --prometheus

sleep 180 # todo - implement a watch

kubectl create ns bookinfo

kubectl label namespace bookinfo istio-injection=enabled --overwrite


kubectl apply -f examples/bookinfo/bookinfo.yaml -ns bookinfo
