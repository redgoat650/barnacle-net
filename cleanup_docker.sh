#!/bin/bash

runningContainers=$(docker container ls --format json | jq -r '. | select(.Image|startswith("redgoat650/barnacle-net")) | .Names' | xargs)

docker stop ${runningContainers}

docker stop barnacle-watchtower