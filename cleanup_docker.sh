#!/bin/bash

runningContainers=$(docker container ls --all --format json | jq -r '. | select(.Image|startswith("redgoat650/barnacle-net")) | .Names' | xargs)

docker stop ${runningContainers}

docker rm ${runningContainers}

docker stop barnacle-watchtower
