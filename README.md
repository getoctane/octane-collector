# Octane Collector
The collector is a gateway to the Octane Ledger service:
* Polling - reads meter configs from the Octane Ledger service, executes meter readings accordingly, and sends results to the service
* Push - allows collocated services to push custom measurements by proxying their requests to the Octane Ledger service

# Installation
A docker image is publicly available here:
us.gcr.io/octane-public/octane-collector

Start a container with the image, setting the appropriate ENV variables:
1. LEDGER_HOST - Octane Legder Service host (i.e., "http://ledger.getoctane.io")
2. CLUSTER_KEY - API Key provided by Octane for new clusters
3. PROMETHEUS_HOST (optional) - Prometheus host name
4. QUEUE_PUSH_INTERVAL_MINS - Frequency with which collector should push measurements to service
5. QUEUE_DIR - Path to directory where collector may store queue of measurements
6. ENABLE_K8S_METRICS_SURVEYOR - Whether the collector should automatically query the kubernetes metrics api for usage metrics

# Usage
## Polling mechanism
Define meters in the Octane Ledger service. Send HTTP POST request to the Octane Ledger service's /vendor/meters endpoint.  The HEADER should include Authorization with the vendor API Key. The body should be JSON with the following format:
```
{
    "meter_name": <name_identifying_meter>,
    "meter_type": <type_of_meter>,
    "meter_value": <query_for_meter>
}
```

At this time, the collector can only poll from Prometheus for meters of type 'prometheus'.

## Push mechanism
Send HTTP POST requests to the collector's /instance/measurements endpoint.  The body should be JSON with the following format:
```
{
    "meter_name": <name_identifying_meter>,
    "measurements": [{"time": <measurment_time>, "value": <measurment_value>}],
    "namespace": <optional_namespace_name>,
    "pod": <optional_pod_name>,
    "labels": <optional_labels>
}
```

Where labels is a dictionary in the format:
```
{
    <label_name>: <label_value>
}
``` 