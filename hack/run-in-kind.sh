#! /bin/bash

set -e -o pipefail

# cd to top dir
scriptdir="$(dirname "$(realpath "$0")")"
cd "$scriptdir/.."


# Make clean
docker rmi `docker images | awk '{print $3}'` --force || true

# Build the container images
make docker-build
make -C downloader image
make -C export image

# Load the images into kind
# We are using a special tag that should never be pushed to a repo so that it's
# obvious if we try to run a container other than these intended ones.
KIND_TAG=latest
IMAGES=(
        "quay.io/octo-emerging/gitops-primer"
        "quay.io/octo-emerging/gitops-primer-export"
        "quay.io/octo-emerging/gitops-primer-downloader"
)
for i in "${IMAGES[@]}"; do
    docker tag "${i}:latest" "${i}:${KIND_TAG}"
    kind load docker-image "${i}:${KIND_TAG}"
done
