#!/bin/bash

export AWS_PAGER=""


ENV=$1
ROOT_DOMAIN=$2
ROOT_DOMAIN_HOSTED_ZONE_ID=$3
ACTION=$4
STACK_NAME=emerald-$ENV

usage() {
    printf "\033[36m==> %s\033[0m\n" "[ERROR] usage: ./attach_hosted_zone.sh <env> <root-domain> <root-domain-hosted-zone-id> <CREATE|DELETE>"
}

if [ -z "$ENV" ] || [ -z "$ROOT_DOMAIN" ] || [ -z "$ROOT_DOMAIN_HOSTED_ZONE_ID" ] || [ -z "$ACTION" ]; then
    usage
    exit 1
fi

if [ "$ACTION" != "CREATE" ] && [ "$ACTION" != "DELETE" ]; then
    usage
    exit 1
fi

aws cloudformation wait stack-exists --stack-name $STACK_NAME
printf "\033[36m==> %s\033[0m\n" "locating loadbalancer arn..."

load_balancer_arn=`aws cloudformation describe-stack-resources --stack-name $STACK_NAME | jq -rc '.StackResources | .[] | select(.LogicalResourceId == "LoadBalancer") .PhysicalResourceId'`

printf "\033[36m==> %s\033[0m\n" "found load balancer arn=$load_balancer_arn"

echo "locating DNS name..."
dns_name=`aws elbv2 describe-load-balancers --load-balancer-arns $load_balancer_arn | jq -rc '.LoadBalancers | .[] | select(.LoadBalancerArn == '\"$load_balancer_arn\"') .DNSName'`
hosted_zone_id=`aws elbv2 describe-load-balancers --load-balancer-arns $load_balancer_arn | jq -rc '.LoadBalancers | .[] | select(.LoadBalancerArn == '\"$load_balancer_arn\"') .CanonicalHostedZoneId'`


printf "\033[36m==> %s\033[0m\n" "found DNS name=$dns_name"

printf "\033[36m==> %s\033[0m\n" "updating 'A' record under $ROOT_DOMAIN pointing $ENV.$ROOT_DOMAIN --> $dns_name under hosted-zone=$hosted_zone_id..."

record_set="/tmp/emld-change-resource-record-set.json"
cat > $record_set << EOF
{
  "Comment": "Updating Alias resource record sets in Route 53",
  "Changes": [
    {
      "Action": "$ACTION",
      "ResourceRecordSet": {
        "Name": "$ENV.$ROOT_DOMAIN",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "$hosted_zone_id",
          "DNSName": "$dns_name",
          "EvaluateTargetHealth": true
        }
      }
    }
  ]
}
EOF

printf "\033[36m==> %s\033[0m\n" "updating record sets..."
aws route53 change-resource-record-sets --hosted-zone-id $ROOT_DOMAIN_HOSTED_ZONE_ID --change-batch file://$record_set
