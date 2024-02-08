# Install Spicedb-operator

# Create Spicedb namespace
```kubectl create namespace spicedb-operator```

# Deploy the Spicedb operator
```
kubectl apply --server-side -f https://github.com/authzed/spicedb-operator/releases/latest/download/bundle.yaml -n spicedb
```

# Create namespace spicedb

```kubectl create namespace spicedb```


