package surveyors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/util"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeNetcSurveyor struct {
	k *kubernetes.Clientset
}

func NewKubeNetcSurveyor(k *kubernetes.Clientset) (*KubeNetcSurveyor, error) {
	return &KubeNetcSurveyor{k}, nil
}

func (s *KubeNetcSurveyor) getKubeNetcMetrics(host string) ([]*ledger.MeasurementList, error) {
	parsed, err := util.PrometheusExporterRequest(host)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	measurementLists := []*ledger.MeasurementList{}
	for _, mf := range parsed {
		metricName := mf.GetName()

		switch metricName {
		case "bytes_sent":
		case "bytes_recv":
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
				labels[labelPair.GetName()] = labelPair.GetValue()
			}

			if strings.HasPrefix(labels["destination_address"], "127.0.0.1") {
				continue
			}
			if strings.HasPrefix(labels["source_address"], "127.0.0.1") {
				continue
			}

			measurmentList := &ledger.MeasurementList{
				MeterName: metricName,
				Labels:    labels,
				Measurements: []*ledger.Measurement{
					&ledger.Measurement{
						Value: value,
						Time:  timestamp,
					},
				},
			}
			measurementLists = append(measurementLists, measurmentList)
		}
	}
	return measurementLists, nil
}

func (s *KubeNetcSurveyor) Survey() ([]*ledger.MeasurementList, error) {
	opts := metav1.ListOptions{
		LabelSelector: "name=kube-netc",
	}
	netcPods, err := s.k.CoreV1().Pods("kube-system").List(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	allResults := []*ledger.MeasurementList{}

	for _, pod := range netcPods.Items {
		result, err := s.getKubeNetcMetrics("http://" + pod.Status.PodIP + ":9655")

		if err != nil {
			fmt.Printf("ERROR: Could not fetch kube-netc metrics from %s -- %s\n", pod.Status.PodIP, err.Error())
			continue
		}
		allResults = append(allResults, result...)
	}

	return allResults, nil
}
