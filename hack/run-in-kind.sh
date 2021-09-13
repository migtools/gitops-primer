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
make -C webhook image

# Load the images into kind
# We are using a special tag that should never be pushed to a repo so that it's
# obvious if we try to run a container other than these intended ones.
IMAGES=(
        "quay.io/konveyor/gitops-primer:latest"
        "quay.io/konveyor/gitops-primer-export:latest"
        "quay.io/konveyor/gitops-primer-downloader:latest"
        "quay.io/konveyor/gitops-primer-webhook:latest"
)
for i in "${IMAGES[@]}"; do
    kind load docker-image "${i}"
done
