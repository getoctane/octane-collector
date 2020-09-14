package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	// Octane settings
	ledgerScheme string
	ledgerHost   string
	clusterKey   string

	queueDir                 string
	queuePushIntervalMinutes uint16

	// Required for Prometheus-type meter queries
	prometheusHost string

	// Only required for testing K8S-based Surveyors in development
	kubeconfig string

	enableK8SMetricsSurveyor bool
	kubeStateMetricsHost     string // Required for surveying kube-state-metrics

	// On a cluster, Pod addresses are discovered dynamically for kube-netc. This
	// setting allows for overriding that behavior -- useful for dev.
	kubeNetcHostsOverride []string
)

func requireEnvVar(varName string) string {
	val := os.Getenv(varName)
	if val == "" {
		panic(fmt.Sprintf("%s environment variable required", varName))
	}
	return val
}

func init() {
	ledgerHost = requireEnvVar("LEDGER_HOST")
	clusterKey = requireEnvVar("CLUSTER_KEY")

	u, err := url.Parse(ledgerHost)
	if err != nil {
		panic(err)
	}
	if u.Host == "" {
		panic(fmt.Sprintf("Cannot parse ledger host value '%s'", ledgerHost))
	}
	ledgerScheme = u.Scheme
	ledgerHost = u.Host + "/" + u.Path

	queueDir = requireEnvVar("QUEUE_DIR")

	queuePushIntervalMinutes = 1 // default

	pushIntervalStr := os.Getenv("QUEUE_PUSH_INTERVAL_MINS")
	if pushIntervalStr != "" {
		parsedInterval, err := strconv.ParseUint(pushIntervalStr, 10, 16)
		if err != nil {
			panic(err)
		}
		queuePushIntervalMinutes = uint16(parsedInterval)
	}
	fmt.Printf("Setting queue push interval to %d minutes\n", queuePushIntervalMinutes)

	prometheusHost = os.Getenv("PROMETHEUS_HOST")

	kubeconfig = os.Getenv("KUBECONFIG")

	enableK8SMetricsSurveyor = os.Getenv("ENABLE_K8S_METRICS_SURVEYOR") == "true"
	kubeStateMetricsHost = os.Getenv("KUBE_STATE_METRICS_HOST")

	kubeNetcHostsOverrideStr := os.Getenv("KUBE_NETC_HOSTS_OVERRIDE")
	kubeNetcHostsOverride = strings.Split(kubeNetcHostsOverrideStr, ",")
}
