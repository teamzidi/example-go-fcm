#!/usr/bin/env bash

set -eu

topic=$1
payload='{"title":"HI, FCM!","body":"from push_topic.sh","topic":"'$topic'"}'

echo payload=$payload

decoded=$(echo -n "$payload" | base64 -w 0)

curl -d '{"message":{"data":"'$decoded'"}}' localhost:8080/publish/topic
