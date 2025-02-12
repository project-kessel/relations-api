# Testing Changes in Ephemeral

## Prerequistes
You'll need:
1) Bonfire
2) Podman/Docker
3) Credentials to login to Quay.io (can be your personal credentials or a [Robot Account](https://docs.quay.io/glossary/robot-accounts.html) if desired)
4) A public Quay repo to push your image to

**tl;dr**
1) Make your changes
2) Build and push you changes in a container image:

```shell
export IMAGE=quay.io/your-quay-repo/kessel-relations-api
export QUAY_USER=your-quay-username
export QUAY_TOKEN=your-quay-password
export RH_REGISTRY_USER=your-redhat-registry-user
export RH_REGISTRY_TOKEN=your-redhat-registry-token
make docker-build-push`
```

3) Update Bonfire CLI

See https://github.com/RedHatInsights/bonfire?tab=readme-ov-file#installing-locally for installing bonfire.

To update from the last time:
```shell
VENV_DIR=~/bonfire_venv
. $VENV_DIR/bin/activate
pip install --upgrade pip
pip install --upgrade crc-bonfire
```

4) Update Bonfire local config for your custom image
5) Deploy to ephemeral with Bonfire


## Setting up a local config for Bonfire
Bonfire allows you to control what is deployed to ephemeral by using a local config. Is it very handy for testing changes in ephemeral without having to get your changes merged first

The default config file is located at $HOME/.config/bonfire/config.yaml

```yaml
# Sample Config for Relations API
apps:
- name: kessel
  components:
    - name: kessel-relations
      host: local
      repo: /path/to/relations-api/code/locally # this is the path to the cloned repo on your system
      path: deploy/kessel-relations.yaml        # this is the path to the deploy file for ephemeral but can be changed to whatever you like
      parameters:                               # parameters equate to parameters defined in the template -- any parameter can be overwritten
        RELATIONS_IMAGE: quay.io/your-quay-repo/kessel-relations-api
        RELATIONS_IMAGE_TAG: your-image-tag-for-above-image
```

To deploy your version of Relations, run `bonfire deploy kessel -C kessel-relations --local-config-method merge`

In the output you'll see where bonfire detects your settings for this app and component and will merge your settings with the settings defined in App Interface

```shell
2025-02-11 10:34:50 [    INFO] [          MainThread] local configuration found for apps: ['kessel']
2025-02-11 10:34:50 [    INFO] [          MainThread] diff in apps config after merging local config into remote config:
```

## Building Container images for testing

Building your own container image to test with is easy, you just need a **public** quay repo to push to and consume from. These images are designed to be consumed in a cluster.

> Note: If your Quay repo is not public, the cluster will not be able to pull the image. By default, when you push an image to Quay for the first time, it will create the repo for you, but it is set to private by default. This can be changed by navigating to the repo in Quay --> Settings --> Repository Visibility --> Make Public

> Another Note: The process is slightly different for Mac to encompass those using ARM laptops. Using the `build-push-minimal` make target ensures the image is built for Linux/AMD64 to ensure it can run on clusters but may not be ideal for running locally for those on ARM systems.


**To build the image on Linux:**
1) Set the image repo for where the image should be pushed to: `export IMAGE=quay.io/my-repo/relations-api`
2) Set your  Quay.io credentials so your container engine can login to push:

```shell
export QUAY_USER=your-quay-username
export QUAY_TOKEN=your-quay-password
export RH_REGISTRY_USER=your-redhat-registry-user
export RH_REGISTRY_TOKEN=your-redhat-registry-token
```

3) Build and push the image: `make docker-build-push`

**On Mac:**
1) Set the image repo for where the image should be pushed to: `export QUAY_REPO_RELATIONS=your-quay-repo`
2) Login to Quay with Podman/Docker and your Quay credentials: `podman login quay.io` or `docker login quay.io`
3) Build and push the image: `make build-push-minimal`

The above will build the container using the same or similar build script used by our build systems to ensure its a prod-like test image. This image can then be plugged into the bonfire config and used to test in ephemeral.
