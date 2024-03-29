#!/bin/bash
# Convenience script for exporting AWS environmental variables

export AWS_SESSION_TOKEN=$(aws sts get-session-token | jq ".Credentials.SessionToken")
export AWS_ACCESS_KEY_ID=$(aws configure get default.aws_access_key_id)
export AWS_SECRET_ACCESS_KEY=$(aws configure get default.aws_secret_access_key)
