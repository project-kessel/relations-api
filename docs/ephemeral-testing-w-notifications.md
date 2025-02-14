# Running Notifications + Relations + Inventory using Ephemeral with Custom Images

This process goes through running Notifications, Inventory API and Relations API in Ephemeral leveraging custom built container images. By using custom built images, you can test code changes in an OpenShift cluster, and reap all the benefits of Clowder to handle dependencies.

## Prerequisites

You'll need the following tools:
* Docker/Podman
* make
* git
* [bonfire](https://github.com/RedHatInsights/bonfire)
* Access to Quay and personal Quay Repos for each service
* VPN Connection to Red Hat Corp VPN (for accessing Ephemeral environment to deploy)

### Bonfire Setup

If you have not used Bonfire before or in a long time, you'll likely need to go through a few setup steps:
* [Installing Locally](https://github.com/RedHatInsights/bonfire/tree/master?tab=readme-ov-file#installing-locally) will walk you through installing bonfire and setting up a Github PAT token needed to use the Github API
* The [Quick Start](https://github.com/RedHatInsights/bonfire/tree/master?tab=readme-ov-file#quick-start) will walk you through some basic commands and use cases
* Reading through the [Local Config](https://github.com/RedHatInsights/bonfire/tree/master?tab=readme-ov-file#using-a-local-config) is also beneficial as this SOP will leverage local configs

To build custom images for these services, you'll also need the following repos cloned to your system:
* [Relations API](https://github.com/project-kessel/relations-api)
* [Inventory API](https://github.com/project-kessel/inventory-api)
* [Notifications Backend](https://github.com/RedHatInsights/notifications-backend)

### Quay Setup

When building your own container images to test, you'll need a **public** quay repo to push to and consume from. You'll need to create a Quay repo for each service.

To create a Repo:
* Access [Quay](https://quay.io/) and login
* Hit the **Create New Repository** button
* Enter a Name for the repo (service name is probably a good idea)
* Select the **public** radio button
* Select the **empty repository** radio button
* Hit **Create Public Repository**

If you intend to build custom images for each service, you'll want a Quay repo for each service (inventory-api, relations-api, notifications-backend).

> NOTE: If you already have a Quay repo for an image, you can make sure its set to public by navigating to the repo in Quay --> Settings --> Repository Visibility --> Make Public

## Building Custom Images

The image build process will require authenticating with Quay and Registry.Redhat.io to push the images. The credentials for Quay and Registry.Redhat.io are generally the same, with username usually being your Red Hat email address. You can test your credentials ahead of time by logging into Quay or using docker/podman login

```shell
# testing Quay
docker/podman login quay.io -u username@redhat.com
Password:
Login Succeeded!

# testing Registry.Redhat.io
docker/podman login registry.redhat.io -u username@redhat.com
Password:
Login Succeeded!
```

> NOTE: The process for Mac and Linux is slightly different to account for those using ARM laptops. The `build-push-minimal` make target used for Mac forces the image be built for linux/amd64 arch to ensure it can run in the ephemeral cluster. If you are running on an ARM laptop, it is not suitable for running locally

### Building Relations API
1) Change to the Relations API code path: `cd /path/to/relations-api`

**On Linux**:

2) Set required ENV vars for the script to run

```shell
export IMAGE=quay.io/you-username/relations-api
export QUAY_USER=your-quay-username
export QUAY_TOKEN=your-quay-password
export RH_REGISTRY_USER=your-redhat-registry-username
export RH_REGISTRY_TOKEN=your-redhat-registry-password
```

3) Build and push the image: `make docker-build-push`

**On Mac:**

2) Set required ENV vars for the script to run:

```shell
export QUAY_REPO_RELATIONS=quay.io/you-username/relations-api
```

3) Login to Quay with Podman/Docker and your Quay credentials: `podman login quay.io` or `docker login quay.io`

4) Build and push the image: `make build-push-minimal`


### Building Inventory API
1) Change to the Inventory API code path: `cd /path/to/inventory-api`

**On Linux**:
2) Set required ENV vars for the script to run

```shell
export IMAGE=quay.io/you-username/inventory-api
export QUAY_USER=your-quay-username
export QUAY_TOKEN=your-quay-password
export RH_REGISTRY_USER=your-redhat-registry-username
export RH_REGISTRY_TOKEN=your-redhat-registry-password
```

3) Build and push the image: `make docker-build-push`

**On Mac:**
2) Set required ENV vars for the script to run:

```shell
export QUAY_REPO_INVENTORY=quay.io/you-username/inventory-api
```

3) Login to Quay with Podman/Docker and your Quay credentials: `podman login quay.io` or `docker login quay.io`

4) Build and push the image: `make build-push-minimal`

### Building Notifications Backend

1) Change to the Notifications Backend code path: `cd /path/to/notifications-backend`

**All Operating Systems**
2) Set ENV vars

```shell
export QUAY_REPO_NOTIFICATIONS=quay.io/username/notifications-backend
export IMAGE_TAG=$(git rev-parse --short=7 HEAD)
```

3) Login to Quay with Podman/Docker and your Quay credentials: `podman login quay.io` or `docker login quay.io`

4) Build and push the image:

```shell
docker/podman build \
  -t ${QUAY_REPO_NOTIFICATIONS}:${IMAGE_TAG} \
  --platform linux/amd64 . \
  -f docker/Dockerfile.notifications-backend.jvm

docker/podman push ${QUAY_REPO_NOTIFICATIONS}:${IMAGE_TAG}
```

## Deploy to Ephemeral (All-in-One)

Since the Notifications Backend ClowdApp defines both Inventory API and Relations API as dependencies, deploying Notifications to ephemeral alone will handle deploying all three. By leveraging custom images and a local bonfire config, you can deploy your custom images for Inventory and/or Relations all while deploying Notifications to ephemeral

> NOTE: If desired, you can deploy/redeploy Inventory API and/or Relations API separately from Notifications. This is covered fairly well in the [Internal Kessel Docs](https://cuddly-tribble-gq7r66v.pages.github.io/start-here/getting-started/)

### Setup Local Config

By default, bonfire ships with a config file located under $HOME/.config/bonfire/config.yaml. This config file can be used for configuring your custom deployments

> NOTE: If your config is missing, or you want to start clean: `bonfire config write-default`

The below local config will allow you to control all three services as needed. Each service specifies some form of image parameter; this is where you can set your custom image that has your code changes.

> NOTE: If you do not have changes to test for a particular service, you can comment out the parameter section for the service, or set it to its respective default values
>
> Notifications Image/Tag Default: quay.io/cloudservices/notifications-backend:latest
>
> Relations Image/Tag Default: quay.io/redhat-services-prod/project-kessel-tenant/kessel-relations/relations-api: latest
>
> Inventory Image/Tag Default: quay.io/redhat-services-prod/project-kessel-tenant/kessel-inventory/inventory-api:latest

Update your local config with the below settings:

```yaml
# $HOME/.config/bonfire/config.yaml
apps:
- name: notifications
  components:
    - name: notifications-backend
      host: local
      repo: /path/to/notifications-backend-cloned-repo
      path: .rhcicd/clowdapp-backend.yaml
      parameters:
        IMAGE: quay.io/username/notifications-backend
        IMAGE_TAG: your-image-tag
        NOTIFICATIONS_KESSEL_INVENTORY_ENABLED: true
        NOTIFICATIONS_KESSEL_RELATIONS_ENABLED: true
- name: kessel
  components:
    - name: kessel-inventory
      host: local
      repo: /path/to/inventory-api-cloned-repo
      path: deploy/kessel-inventory-ephem.yaml
      parameters:
        INVENTORY_IMAGE: quay.io/username/inventory-api
        IMAGE_TAG: your-image-tag
    - name: kessel-relations
      host: local
      repo: /path/to/relations-api-cloned-repo
      path: deploy/kessel-relations.yaml
      parameters:
        RELATIONS_IMAGE: quay.io/username/relations-api
        RELATIONS_IMAGE_TAG: latest
```

### Deploy

After updating your config, ensuring your images are built, and publiclly available in Quay, run the following to deploy all 3 services

```shell
bonfire deploy notifications --source=appsre --ref-env insights-stage --timeout 600 --local-config-method merge
```

The above deployment can take some time as it spins up all notifications services plus Kessel services. You can check the state of the deployment by investigating the different pods in Console or terminal (`oc get pods`).
