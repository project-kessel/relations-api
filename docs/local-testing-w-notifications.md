# Running Notifications + Relations + Inventory using Local Built Binaries

### Running Relations:
```shell
# Start up SpiceDB Alt -- uses a different postgres port to avoid conflicts with notifications
make spicedb-alt-up

# Start relations
make run
```

### Running Inventory:
The process to run Inventory locally can be found in Inventory API's [README](https://github.com/project-kessel/inventory-api?tab=readme-ov-file#kessel-inventory--kessel-relations-using-built-binaries)

By default, Inventory will leverage a SQLite database and create a local db file called `inventory.db`. If you wish to use postgres, you'll need a postgres database running and the config file used in the above doc would need to be updated. An example of configuring Inventory API for postgres can be found [HERE](https://github.com/project-kessel/inventory-api/blob/b19bc4cef8570b8e34f85336067a0b48f9dcf910/inventory-api-compose.yaml#L19)

### Running Notifications

> NOTE: During the clean and install step tests are run that may not work if you do not have Docker -- YMMV

```shell
# Spin up the Notifications DB
podman run --name notifications_db --detach -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=notifications -p 5432:5432 docker.io/postgres:latest -c log_statement=all

# Clean, Compile, Test, and Install
./mvnw clean install

# OR to skip tests (takes about 15 mins)
./mvnw clean install -Dmaven.test.skip

# Run the Notifications Service
./mvnw clean quarkus:dev -Dnotifications.use-default-template=true -Dnotifications.kessel-inventory.enabled=true -Dnotifications.kessel-relations.enabled=true -pl :notifications-backend
```

### Cleanup!
```shell
# To kill notifications, enter `q` in the running window to shut it down
# To kill inventory and relations, use whatever fun killing technique you like!

# Teardown SpiceDB
make spicedb-down

# Teardown Notifications DB
podman stop notifications_db && podman rm notifications_db
```
