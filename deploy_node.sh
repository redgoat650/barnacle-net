#!/bin/bash

CONTAINER_NAME=${1:-"barnacle-1"}
SERVER=${2:-"docker"}
TAG=${3:-"scratch"}
INTERVAL=${4:-"30"}

if [ "${SERVER}" = "docker" ]; then
    serverDID=$(docker container ls | grep "barnacle-server" | awk '{ print $1 }')
    echo "Server docker ID: ${serverDID}"

    servAddr=$(docker container inspect $serverDID | jq -r .[0].NetworkSettings.IPAddress)
    echo "Server container IPAddr: ${servAddr}"

    SERVER="${servAddr}:8080"
fi

docker run -d \
  --name ${CONTAINER_NAME} \
  --rm \
  redgoat650/barnacle-net:${TAG} \
  barnacle start --server=${SERVER}

runningContainers=$(docker container ls --format json | jq -r '. | select(.Image|startswith("redgoat650/barnacle-net")) | .Names' | xargs)

docker run -d \
  --name barnacle-watchtower \
  --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --interval=${INTERVAL} \
  ${runningContainers}
