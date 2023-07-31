#!/bin/bash

CONTAINER_NAME=${1:-"barnacle-server"}
TAG=${2:-"scratch"}
INTERVAL=${3:-"30"}

docker run -d \
  --name ${CONTAINER_NAME} \
  --rm \
  -p 8080:8080 \
  redgoat650/barnacle-net:${TAG} \
  server start

# docker run -d \
#   --name server-watchtower \
#   --rm \
#   -v /var/run/docker.sock:/var/run/docker.sock \
#   containrrr/watchtower \
#   --interval=${INTERVAL} \
#   ${CONTAINER_NAME}
