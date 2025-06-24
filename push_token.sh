#!/usr/bin/env bash

set -eu

token=$1
payload='{"title":"hi, token!","body":"from push_token.sh","token":"'$token'"}'

echo payload=$payload

decoded=$(echo -n "$payload" | base64 -w 0)

curl -d '{"message":{"data":"'$decoded'"}}' localhost:8080/publish/token
