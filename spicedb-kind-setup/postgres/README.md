`kubectl create namespace spicedb`

`kubectl apply -f secret.yaml -n spicedb`
`kubectl apply -f storage.yaml -n spicedb`
`kubectl apply -f postgresql.yaml -n spicedb`