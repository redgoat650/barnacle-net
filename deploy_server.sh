#!/bin/bash

CONTAINER_NAME=${1:-"barnacle-server"}
TAG=${2:-"scratch"}
INTERVAL=${3:-"30"}

docker stop barnacle-watchtower

docker run -d \
  --name ${CONTAINER_NAME} \
  --rm \
  -p 8080:8080 \
  --ip "172.17.0.4" \
  redgoat650/barnacle-net:${TAG} \
  server start

runningContainers=$(docker container ls --format json | jq -r '. | select(.Image|startswith("redgoat650/barnacle-net")) | .Names' | xargs)

docker run -d \
  --name barnacle-watchtower \
  --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --interval ${INTERVAL} \
  ${runningContainers}
