#!/bin/bash

TAG=${1:-"scratch"}
BUILD_BASE=${2:-""}

if [ -n "$BUILD_BASE" ]; then
    echo "building base"

    pushd ./docker/base
    docker buildx build \
        -t redgoat650/barnacle-net:${TAG}-base \
        --push \
        .
        
    docker buildx build \
        -t redgoat650/barnacle-net:${TAG}-base \
        --platform linux/arm/v6 \
        --push \
        .
    popd
fi

docker buildx create --use --name=xplat --node=crossplat &&
docker buildx build \
    --platform linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64 \
    --push \
    --tag redgoat650/barnacle-net:${TAG} \
    .


# # Build the app
# go build
# mv ./barnacle-net ./docker/barnacle-net

# pushd ./docker

# docker buildx build \
#     -t redgoat650/barnacle-net:${TAG} \
#     --push \
#     .

# rm ./barnacle-net
# popd

# # Build the app armv6
# env GOOS=linux GOARCH=arm GOARM=6 go build 
# mv ./barnacle-net ./docker/barnacle-net

# pushd ./docker

# docker buildx build \
#     -t redgoat650/barnacle-net:${TAG} \
#     --platform linux/arm/v6 \
#     --push \
#     .

# rm ./barnacle-net
# popd
