#!/bin/bash

TAG=${1:-"scratch"}

# Install the app
go install

pushd ./docker
cp /go/bin/barnacle-net ./barnacle-net
docker build -t redgoat650/barnacle-net:${TAG} .
rm ./barnacle-net
popd

docker push redgoat650/barnacle-net:${TAG}