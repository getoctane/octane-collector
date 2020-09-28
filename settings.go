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

	surveyingIntervalMinutes uint16
	meteringIntervalMinutes  uint16

	// Required for Prometheus-type meter queries
	prometheusHost string

	// Only required for testing K8S-based Surveyors in development
	kubeconfig string

	enableSurveyors      bool
	kubeStateMetricsHost string // Required for surveying kube-state-metrics

	// On a cluster, Pod addresses are discovered dynamically for kube-netc. This
	// setting allows for overriding that behavior -- useful for dev.
	kubeNetcNamespaceOverride string
	kubeNetcHostsOverride     []string
)

func requireEnvVar(varName string) string {
	val := os.Getenv(varName)
	if val == "" {
		panic(fmt.Sprintf("%s environment variable required", varName))
	}
	return val
}

func parseIntSettingWithDefault(varName string, def uint16) uint16 {
	value := def
	str := os.Getenv(varName)
	if str != "" {
		parsed, err := strconv.ParseUint(str, 10, 16)
		if err != nil {
			panic(err)
		}
		value = uint16(parsed)
	}
	return value
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

	queuePushIntervalMinutes = parseIntSettingWithDefault("QUEUE_PUSH_INTERVAL_MINS", 1)
	fmt.Printf("Setting queuePushIntervalMinutes to %d\n", queuePushIntervalMinutes)

	// These will both default to the push interval minutes
	surveyingIntervalMinutes = parseIntSettingWithDefault("SURVEYING_INTERVAL_MINS", queuePushIntervalMinutes)
	fmt.Printf("Setting surveyingIntervalMinutes to %d\n", surveyingIntervalMinutes)

	meteringIntervalMinutes = parseIntSettingWithDefault("METERING_INTERVAL_MINS", queuePushIntervalMinutes)
	fmt.Printf("Setting meteringIntervalMinutes to %d\n", meteringIntervalMinutes)

	prometheusHost = os.Getenv("PROMETHEUS_HOST")

	kubeconfig = os.Getenv("KUBECONFIG")

	// Default true
	enableSurveyors = os.Getenv("ENABLE_SURVEYORS") != "false"
	kubeStateMetricsHost = os.Getenv("KUBE_STATE_METRICS_HOST")

	kubeNetcNamespaceOverride = os.Getenv("KUBE_NETC_NAMESPACE_OVERRIDE")
	kubeNetcHostsOverrideStr := os.Getenv("KUBE_NETC_HOSTS_OVERRIDE")
	splitFn := func(c rune) bool {
		return c == ','
	}
	kubeNetcHostsOverride = strings.FieldsFunc(kubeNetcHostsOverrideStr, splitFn)
}
