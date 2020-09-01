package surveyors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/util"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeNetcSurveyor struct {
	k            *kubernetes.Clientset
	hostOverride string
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

func NewKubeNetcSurveyor(k *kubernetes.Clientset, hostOverride string) (*KubeNetcSurveyor, error) {
	return &KubeNetcSurveyor{k, hostOverride}, nil
}

func (s *KubeNetcSurveyor) getKubeNetcMetrics(host string, nodes *v1.NodeList) ([]*ledger.MeasurementList, error) {
	parsed, err := util.PrometheusExporterRequest(host)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	nodeZones := make(map[string]string)
	for _, node := range nodes.Items {
		nodeZones[node.Name] = node.Labels["topology.kubernetes.io/zone"]
	}

	podDataTransferInfo := make(map[string]*dataTransferMeasurements)

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
			if labels["destination_kind"] == "node" { // Don't want to see traffic to Nodes (TODO ?)
				continue
			}
			if strings.HasPrefix(labels["destination_address"], "127.0.0.1") {
				continue
			}
			if strings.HasPrefix(labels["source_address"], "127.0.0.1") {
				continue
			}

			podKey := labels["source_namespace"] + "/" + labels["source_name"]

			if _, ok := podDataTransferInfo[podKey]; !ok {
				podDataTransferInfo[podKey] = newDataTransferMeasurements(labels["source_namespace"], labels["source_name"], timestamp)
			}

			switch labels["destination_kind"] {
			case "pod":

				srcZone, ok := nodeZones[labels["source_node"]]
				if !ok {
					fmt.Printf("Can't find source zone for node %s?\n", labels["source_node"])
					continue
				}
				dstZone, ok := nodeZones[labels["destination_node"]]
				if !ok {
					fmt.Printf("Can't find destination zone for node %s?\n", labels["destination_node"])
					continue
				}

				if srcZone == dstZone {
					podDataTransferInfo[podKey].intraZoneEgress.Measurements[0].Value += value
				} else {
					podDataTransferInfo[podKey].interZoneEgress.Measurements[0].Value += value
				}

			case "":

				podDataTransferInfo[podKey].internetEgress.Measurements[0].Value += value

			default:
				continue
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
	allResults := []*ledger.MeasurementList{}

	if s.hostOverride == "" {
		opts := metav1.ListOptions{
			LabelSelector: "name=kube-netc",
		}
		netcPods, err := s.k.CoreV1().Pods("kube-system").List(context.Background(), opts)
		if err != nil {
			return nil, err
		}

		for _, pod := range netcPods.Items {
			result, err := s.getKubeNetcMetrics("http://"+pod.Status.PodIP+":9655", nodes)

			if err != nil {
				fmt.Printf("ERROR: Could not fetch kube-netc metrics from %s -- %s\n", pod.Status.PodIP, err.Error())
				continue
			}
			allResults = append(allResults, result...)
		}
	} else {

		result, err := s.getKubeNetcMetrics(s.hostOverride, nodes)
		if err != nil {
			return nil, fmt.Errorf("ERROR: Could not fetch kube-netc metrics from %s -- %s\n", s.hostOverride, err.Error())
		}
		allResults = append(allResults, result...)
	}

	return allResults, nil
}
