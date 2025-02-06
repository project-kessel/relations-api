# Publish relations images to your personal quay.io repository
# Compatible with MacOS and other archs for cross-compilation
# Excludes redhat.registry.io as this is not needed for local/ephem development
set -exv

if [[ -z "$QUAY_REPO_RELATIONS" ]]; then
    # required since this script is not used in the CI pipeline, publishing should
    # only happen from a developer's local machine to their personal repo
    echo "QUAY_REPO_RELATIONS must be set"
    exit 1
fi
IMAGE_TAG=$(git rev-parse --short=7 HEAD)

source ./scripts/check_docker_podman.sh
${DOCKER} build --platform linux/amd64 --build-arg TARGETARCH=amd64 -t "${QUAY_REPO_RELATIONS}:${IMAGE_TAG}" -f ./Dockerfile
${DOCKER} push "${QUAY_REPO_RELATIONS}:${IMAGE_TAG}"
