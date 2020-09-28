package surveyors

import (
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/util"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type KubeStateMetricsSurveyor struct {
	host string
}

func NewKubeStateMetricsSurveyor(host string) (*KubeStateMetricsSurveyor, error) {
	return &KubeStateMetricsSurveyor{host}, nil
}

func (s *KubeStateMetricsSurveyor) Survey() ([]*ledger.MeasurementList, error) {
	parsed, err := util.PrometheusExporterRequest(s.host)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	measurementLists := []*ledger.MeasurementList{}
	for _, mf := range parsed {
		metricName := mf.GetName()
		for _, m := range mf.GetMetric() {
			var namespace string = ""
			var pod string = ""

			labels := make(map[string]string)
			for _, label := range m.GetLabel() {
				name := label.GetName()
				switch name {
				case "namespace":
					namespace = label.GetValue()
				case "pod":
					pod = label.GetValue()
				default:
					labels[label.GetName()] = label.GetValue()
				}
			}

			if value := m.GetGauge().GetValue(); value != 0 {
				measurementList := &ledger.MeasurementList{
					Namespace: namespace,
					Pod:       pod,
					Labels:    labels,
					Measurements: []*ledger.Measurement{
						&ledger.Measurement{
							MeterName: metricName,
							Value:     value,
							Time:      timestamp,
						},
					},
				}
				measurementLists = append(measurementLists, measurementList)
			}
		}
	}

	return measurementLists, nil
}
