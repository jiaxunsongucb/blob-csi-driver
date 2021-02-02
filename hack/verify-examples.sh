#!/bin/bash
# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

echo "begin to create deployment examples ..."
if [[ "$1" == "azurestackcloud" ]]; then
    STORAGECLASS_FILE="deploy/example/storageclass-blobfuse-azurestack.yaml"
    sed -i "s/AZURE_STACK_STORAGE_ENDPOINT_SUFFIX/$2/g" $STORAGECLASS_FILE
    kubectl apply -f $STORAGECLASS_FILE
else
    kubectl apply -f deploy/example/storageclass-blobfuse.yaml
fi

kubectl apply -f deploy/example/deployment.yaml
kubectl apply -f deploy/example/statefulset.yaml
kubectl apply -f deploy/example/statefulset-nonroot.yaml

echo "sleep 60s ..."
sleep 60

kubectl get pods --field-selector status.phase=Running | grep deployment-blob
kubectl get pods --field-selector status.phase=Running | grep statefulset-blob-0
kubectl get pods --field-selector status.phase=Running | grep statefulset-blob-nonroot-0

echo "deployment examples running completed."
