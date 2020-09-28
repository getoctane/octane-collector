package surveyors

import (
	"context"
	"time"

	"github.com/getoctane/octane-collector/ledger"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"

	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type MetricsServerSurveyor struct {
	km *metricsv.Clientset
}

func NewMetricsServerSurveyor(cfg *rest.Config) (*MetricsServerSurveyor, error) {
	km, err := metricsv.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &MetricsServerSurveyor{km}, nil
}

func (s *MetricsServerSurveyor) Survey() ([]*ledger.MeasurementList, error) {
	podMetricsList, err := s.km.MetricsV1beta1().PodMetricses("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	measurementLists := []*ledger.MeasurementList{}

	for _, pm := range podMetricsList.Items {
		for _, container := range pm.Containers {

			cpuMillicores := float64(container.Usage.Cpu().ScaledValue(resource.Milli))
			memoryBytes := float64(container.Usage.Memory().ScaledValue(0))

			labels := make(map[string]string)
			if pm.Labels != nil {
				for k, v := range pm.Labels {
					labels[k] = v
				}
			}
			labels["container_name"] = container.Name

			ml := &ledger.MeasurementList{
				Namespace: pm.Namespace,
				Pod:       pm.Name,
				Labels:    labels,
				Measurements: []*ledger.Measurement{
					&ledger.Measurement{
						MeterName: "k8s_cpu_milli",
						Value:     cpuMillicores,
						Time:      timestamp,
					},
					&ledger.Measurement{
						MeterName: "k8s_mem_bytes",
						Value:     memoryBytes,
						Time:      timestamp,
					},
				},
			}

			measurementLists = append(measurementLists, ml)
		}
	}

	return measurementLists, nil
}
