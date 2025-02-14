# Running Notifications + Relations + Inventory using Local Built Binaries

This process goes through running Notifications, Inventory API and Relations API locally using built binaries. Since there are database dependencies involved, podman/docker are leveraged to handle running databases.

## Prerequisites

You'll need the following tools:
* Docker/Podman
* make
* git
* [Maven](https://maven.apache.org/install.html)

You'll also need the following repos cloned to your system:
* [Relations API](https://github.com/project-kessel/relations-api)
* [Inventory API](https://github.com/project-kessel/inventory-api)
* [Notifications Backend](https://github.com/RedHatInsights/notifications-backend)

### Running Relations:

1) Change to the Relations API code path: `cd /path/to/relations-api`

2) Start up SpiceDB

```shell
# Start up SpiceDB Alt -- uses a different postgres port to avoid conflicts with notifications
make spicedb-alt-up
```

3) Build and Run Relations API: `make run`

When done, SpiceDB and Postgres should be running in Podman/Docker, and Relations will be running in your locally attached to your terminal

### Running Inventory:
1) Change to the Inventory API code path: `cd /path/to/inventory-api`

2) Build the Inventory API binary: `make local-build`

3) Create and setup the database:

```shell
# default DB is SQLite, removing the file ensures a new database is created each time
rm ./inventory.db
make migrate
```

4) Run Inventory API:

```shell
# The config provided assumes Relations is running locally on ports 8000/9000
# so Inventory is set to use ports 8081/9081 instead to not conflict
./bin/inventory-api serve --config config/inventory-w-relations.yaml
```

### Running Notifications

1) Change to the Notifications Backend code path: `cd /path/to/notifications-backend`
2) Create the notifications DB using Podman/Docker

```shell
podman/docker run --name notifications_db --detach -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=notifications -p 5432:5432 docker.io/postgres:latest -c log_statement=all
```

3) Use Maven to Clean, Compile, Test, and Install

```shell
cd common
mvn clean install
cd ..
./mvnw install

# OR to skip tests (they take about 15 mins)
./mvnw install -Dmaven.test.skip
```

4) Run the Notifications Service

```shell
# If you are not running one of the below kessel services, you can set the enabled flags for that service to false below
./mvnw clean quarkus:dev -Dnotifications.use-default-template=true -Dnotifications.kessel-inventory.enabled=true -Dnotifications.kessel-relations.enabled=true -pl :notifications-backend
```


### Cleanup!

1) Shutdown Notifications: enter `q` in the running window to shut it down

2) Kill inventory and relations: `ctrl+c` or use whatever fun killing technique you like!

3) Teardown SpiceDB:

```shell
cd /path/to/relations-api
make spicedb-down
```

4) Teardown Notifications DB: `podman/docker stop notifications_db && podman/docker rm notifications_db`
