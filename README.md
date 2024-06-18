# Kessel Relations API

Kessel Relations API is a Zanzibar-inspired frontend for relationship-based access control.

## Contributing

### Prerequisites

To get necessary build and generation dependencies:

```
make init
```

See also:

- https://protobuf.dev/getting-started/gotutorial/#compiling-protocol-buffers
- https://go-kratos.dev/en/docs/getting-started/start/

### Build

`make build`

(Configs must be specified to run binary, e.g. `./bin/relations-api -conf configs`, or run make target, below.)

### Run

`make run`

### Create a service

```
# Create a template project
kratos new server

cd server
# Add a proto template
kratos proto add api/server/server.proto
# Generate the proto code
kratos proto client api/server/server.proto
# Generate the source code of service by proto file
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```

### Generate other auxiliary files by Makefile

```
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api

# Generate config code
make config

# Generate all files
make all
```

### Automated Initialization (wire)

```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Spicedb using docker/podman

### Run spicedb and postgresql db with docker/podman compose

`make spicedb`

This is a good option for keeping spicedb running in the background while the rebac service is run via
`make run`, the binary or via the IDE (run/debug) during local development.

### Run the relations-api and spicedb with docker/podman compose

`make rebac`

This runs everything and is a good option for testing a built rebac container image with the running binary.

### Teardown spicedb and postgresql db (brought up with docker/podman compose, as above)

`make spicedb/teardown`

### Teardown rebac and dependencies (brought up with docker/podman compose, as above)

`make rebac/teardown`

### Deploy Rebac and Spicedb using kind/kubernetes

`make kind/rebac`

### Docker

```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/your/configs>:/data/conf <your-docker-image-name>
```

## Deploy to a openshift cluster that has Clowder

### Prerequisite

[bonfire](https://github.com/RedHatInsights/bonfire)

NOTE: The minimum required version of [bonfire](https://github.com/RedHatInsights/bonfire)
is specified in the MIN_BONFIRE_VERSION variable in the deploy.sh script
Bonfire could be upgraded by command:

```asciidoc
pip install --upgrade crc-bonfire
```

Latest version of [bonfire](https://github.com/RedHatInsights/bonfire) could be found [here](https://github.com/RedHatInsights/bonfire/releases).

[oc](https://docs.openshift.com/container-platform/4.8/cli_reference/openshift_cli/getting-started-cli.html)

You should have logged into a valid openshift cluster using the oc login command

`oc login --token=<token> --server=<openshift server>`

### Deploying the components

#### Option 1: Using the config from app-interface (RedHat Internal only)

* Create a config map inside a local directory. You can refer to [here](https://github.com/project-kessel/relations-api/blob/main/deploy/schema-configmaps/ex-schema-configmap.yaml)
  * NOTE: Please replace the "data" section, as per your needs (the schema portion)
* Run the following command:
  * `bonfire deploy relations --import-configmaps  --configmaps-dir <path-to-configmaps-dir>` 

#### Option 2: Using the deploy.sh script in this repository

Note: the deploy script assumes you have a valid oc login and the necessary tools are in place.

The [deploy script](deploy/deploy.sh) under the [deploy](deploy) folder, will deploy all the needed components.

`./deploy.sh`

- Creates a postgres pod and service (Note: No PVC)
- Creates a spiceDB secret - that contains: a preshared key and Postgres connection URI
- Creates a Configmap object - that serves as a bootstrap schema for spiceDB (by default it uses the schema.yaml file under deploy)
- Creates the spiceDB service
- Creates the relations service

You should be able to use the public route (relations-\*) created by the clowder in your namespace, to use the service.

#### Deploying the components with rbac

This is demonstrating calling relationship api from rbac service in ephemeral environment.

```
./deploy.sh rbac <path_to_local_copy_of_insights_rbac>
```

`path_to_local_copy_of_insights_rbac` is this [repository](https://github.com/RedHatInsights/insights-rbac)

Example:

```
./deploy.sh rbac /Projects/insights-rbac
```

- Updates config bonfire file and add rbac component
- Deploys rbac together with relationships application
  - Hardcoded image is used with grpc client for calling relationships
