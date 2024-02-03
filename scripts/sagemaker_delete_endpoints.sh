#!/bin/bash

export AWS_PAGER=""

endpoints=`aws sagemaker list-endpoints | jq .Endpoints | jq -r '.[].EndpointName'`

for endpoint in $endpoints; do
    printf "deleting endpoint $endpoint\n"
    aws sagemaker delete-endpoint --endpoint-name $endpoint
done

endpoint_configs=`aws sagemaker list-endpoint-configs | jq .EndpointConfigs | jq -r '.[].EndpointConfigName'`

for endpoint_config in $endpoint_configs; do
    printf "deleting endpoint config $endpoint_config\n"
    aws sagemaker delete-endpoint-config --endpoint-config-name $endpoint_config
done
