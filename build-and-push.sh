#!/bin/bash
if [ "$#" -ne 1 ]; then
    echo "call with version tag i.e v1.2.3"
    exit 1
fi

make

TAG=$1
HASH=$(echo $HOSTNAME-$(pwd) | shasum -a 256 | cut -c1-8)

docker tag build-${HASH}/provider-bitbucketserver-amd64 docker.io/vinkel/provider-bitbucketserver-amd64:$TAG
docker tag build-${HASH}/provider-bitbucketserver-amd64 docker.io/vinkel/provider-bitbucketserver-amd64:latest
docker tag build-${HASH}/provider-bitbucketserver-controller-amd64 docker.io/vinkel/provider-bitbucketserver-controller-amd64:$TAG
docker tag build-${HASH}/provider-bitbucketserver-controller-amd64 docker.io/vinkel/provider-bitbucketserver-controller-amd64:latest

docker push docker.io/vinkel/provider-bitbucketserver-amd64:$TAG
docker push docker.io/vinkel/provider-bitbucketserver-amd64:latest
docker push docker.io/vinkel/provider-bitbucketserver-controller-amd64:$TAG
docker push docker.io/vinkel/provider-bitbucketserver-controller-amd64:latest