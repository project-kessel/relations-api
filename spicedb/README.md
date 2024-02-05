# Running SpiceDB with PostgreSQL using docker compose

### Required
* using docker/docker compose


### SpiceDB settings:
*  pre-shared key: [env](.env)
*  dashboard address: http://localhost:8080
*   metrics address: http://localhost:9090
*   grpc address: http://localhost:50051

## Postgres settings:

Set the database name, user, password in the secrets folders

*   user: [db.user](secrets/db.user)
*   password: [db.password](secrets/db.password)
* datapase name: [db.name](secrets/db.name)
*   port: 5432

# Start reading PostgreSQL

`./start-postgresql.sh`

# Start SpiceDB 


`docker-compose --env-file .env  up -d`



# Test the Spicedb 

open in the browser `http://localhost:8080/`

## Follow the instructions using zed client

`zed context set first-dev-context :50051 "foobar" --insecure `

## Read schema
`zed --insecure --endpoint=localhost:50051 --token=foobar schema read`