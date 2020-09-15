package surveyors

import (
	"context"
	"fmt"
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/util"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	io_prometheus_client "github.com/prometheus/client_model/go"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeNetcSurveyor struct {
	k             *kubernetes.Clientset
	hostsOverride []string
}

type dataTransferMeasurements struct {
	intraZoneEgress *ledger.MeasurementList
	interZoneEgress *ledger.MeasurementList
	internetEgress  *ledger.MeasurementList
}

func newMeasurementListForPod(meterName string, namespace string, pod string, timestamp string) *ledger.MeasurementList {
	return &ledger.MeasurementList{
		MeterName: meterName,
		Namespace: namespace,
		Pod:       pod,
		Measurements: []*ledger.Measurement{
			&ledger.Measurement{
				Time: timestamp,
			},
		},
	}
}

func newDataTransferMeasurements(namespace string, pod string, timestamp string) *dataTransferMeasurements {
	return &dataTransferMeasurements{
		intraZoneEgress: newMeasurementListForPod("network_egress_intra_zone_bytes", namespace, pod, timestamp),
		interZoneEgress: newMeasurementListForPod("network_egress_inter_zone_bytes", namespace, pod, timestamp),
		internetEgress:  newMeasurementListForPod("network_egress_internet_bytes", namespace, pod, timestamp),
	}
}

func NewKubeNetcSurveyor(k *kubernetes.Clientset, hostsOverride []string) (*KubeNetcSurveyor, error) {
	return &KubeNetcSurveyor{k, hostsOverride}, nil
}

type parsedPromMetrics map[string]*io_prometheus_client.MetricFamily

func fetchNetcMetrics(host string) (parsedPromMetrics, error) {
	parsed, err := util.PrometheusExporterRequest(host)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (s *KubeNetcSurveyor) extrapolateNetcMetrics(ppms []parsedPromMetrics, _ *v1.NodeList) ([]*ledger.MeasurementList, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	podDataTransferInfo := make(map[string]*dataTransferMeasurements)

	for _, parsed := range ppms {
		for _, mf := range parsed {
			metricName := mf.GetName()

			switch metricName {
			case "bytes_sent":
			// case "bytes_recv": -------------- only doing egress for now
			default:
				continue
			}
			for _, m := range mf.GetMetric() {

				value := m.GetGauge().GetValue()

				if value == 0 {
					continue
				}

				labels := make(map[string]string)
				for _, labelPair := range m.GetLabel() {
					// Only pass labels with values... otherwise cr@p ton of data
					if val := labelPair.GetValue(); val != "" {
						labels[labelPair.GetName()] = val
					}
				}

				if labels["source_kind"] != "pod" { // Only want Pods as source
					continue
				}

				podKey := labels["source_namespace"] + "/" + labels["source_name"]

				if _, ok := podDataTransferInfo[podKey]; !ok {
					podDataTransferInfo[podKey] = newDataTransferMeasurements(labels["source_namespace"], labels["source_name"], timestamp)
				}

				switch labels["traffic_type"] {
				case "intra_zone":
					podDataTransferInfo[podKey].intraZoneEgress.Measurements[0].Value += value
				case "inter_zone":
					podDataTransferInfo[podKey].interZoneEgress.Measurements[0].Value += value
				case "internet":
					podDataTransferInfo[podKey].internetEgress.Measurements[0].Value += value
				default:
					fmt.Printf("ERROR: Could not interpret traffic type %s\n", labels["traffic_type"])
					continue
				}

			}
		}
	}

	measurementLists := []*ledger.MeasurementList{}
	for _, dti := range podDataTransferInfo {
		if dti.intraZoneEgress.Measurements[0].Value != 0 {
			measurementLists = append(measurementLists, dti.intraZoneEgress)
		}
		if dti.interZoneEgress.Measurements[0].Value != 0 {
			measurementLists = append(measurementLists, dti.interZoneEgress)
		}
		if dti.internetEgress.Measurements[0].Value != 0 {
			measurementLists = append(measurementLists, dti.internetEgress)
		}
	}

	return measurementLists, nil
}

func (s *KubeNetcSurveyor) Survey(nodes *v1.NodeList) ([]*ledger.MeasurementList, error) {
	hosts := s.hostsOverride
	if len(hosts) == 0 {
		opts := metav1.ListOptions{
			LabelSelector: "name=kube-netc",
		}
		netcPods, err := s.k.CoreV1().Pods("kube-system").List(context.Background(), opts)
		if err != nil {
			return nil, err
		}
		hosts = make([]string, len(netcPods.Items))

		for i, pod := range netcPods.Items {
			hosts[i] = "http://" + pod.Status.PodIP + ":9655"
		}
	}

	ppms := make([]parsedPromMetrics, len(hosts))

	for i, host := range hosts {
		ppm, err := fetchNetcMetrics(host)
		if err != nil {
			fmt.Printf("ERROR: Could not fetch kube-netc metrics from %s -- %s\n", host, err.Error())
			continue
		}
		ppms[i] = ppm
	}

	return s.extrapolateNetcMetrics(ppms, nodes)
}
