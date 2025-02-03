# DB-Configurator
As part of FedRAMP CONMON, all databases are scanned for vulnerabilities. DB-Configurator is a Kubernetes Job that is deployed to FedRAMP clusters to ensure the Relations API database is configured with a user dedicated to performing scans with security tools (currently Nessus).


# How to Deploy in Non-Prod
DB Configurator can and should be deployed via App Interface, but for testing in ephemeral or FedRAMP Int, its possible to deploy this manually with proper access.

**To Deploy**

1) Login to the correct cluster

2) Target the Kessel namespace (or your ephemeral namespace): `oc project kessel-<ENV>`

3) Determine the newest image tag available in [Quay](https://quay.io/repository/app-sre/db-configurator?tab=tags&tag=latest)

4) Create a Secret that contains the desired password for the scan user:

```shell
oc create secret generic nessus-scan-creds --from-literal=password=<INSERT-PASSWORD-HERE>
```

4) Create the Kubernetes Job:

```shell
oc process --local -f deploy-job.yml -p DB_CONFIGURATOR_TAG=<Latest Image Tag> | oc apply -f -
```

This will create a job that spins up a pod to execute the the SQL statement. The job is fully idempotant and can be run multiple times, as the SQL statement executed accounts for the user potentially already existing and having the role.
