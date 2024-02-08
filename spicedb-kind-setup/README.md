# Setup Spicedb-operator with Postgres in local kind kubernetes with monitoring stack

# Run the setup
`./setup.sh`

## Testing grpc end-point
`grpcurl -plaintext spicedb-grpc.127.0.0.1.nip.io:80 list`
```# authzed.api.v1.ExperimentalService
# authzed.api.v1.PermissionsService
# authzed.api.v1.SchemaService
# authzed.api.v1.WatchService
# grpc.health.v1.Health
# grpc.reflection.v1alpha.ServerReflection