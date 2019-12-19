# Event Exporter

This tool is used to export Kubernetes events. It effectively runs a watch on
the apiserver, detecting as granular as possible all changes to the event
objects. Event exporter exports only to CloudWatch Logs.

## Build

To build the binary, run

```shell
make build
```

To run unit tests, run

```shell
make test
```

To build the container, run

```shell
make container
```

## Run

Event exporter require following environment variables:

```
CW_LOG_GROUP_NAME string
CW_LOG_STREAM_NAME string 
AWS_REGION string
```

## Deploy

```
export REGION="us-west-2"
cat https://raw.githubusercontent.com/nithu0115/event-exporter/master/yaml/event-exporter.yaml | sed -e "s/REGION/$REGION/g" |kubectl apply -f -
```

## Notes
### ClusterRoleBinding
This pod's service account should be authorized to get events, you
might need to set up ClusterRoleBinding in order to make it possible. Complete
example with the service account and the cluster role binding you can find in
the `yaml` directory.

### "resourceVersion for the provided watch is too old"
On a system with few/no events, you may see "The resourceVersion for the provided
watch is too old" warnings. These can be ignored. This is due to compacted resource
versions being referenced.