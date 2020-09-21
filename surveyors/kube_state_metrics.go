package surveyors

import (
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/util"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K8SMetricsSurveyor struct {
	k   *kubernetes.Clientset
	km  *metricsv.Clientset
	ksm string
}

func NewK8SMetricsSurveyor(cfg *rest.Config, k *kubernetes.Clientset, ksm string) (*K8SMetricsSurveyor, error) {
	km, err := metricsv.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &K8SMetricsSurveyor{k, km, ksm}, nil
}

// func (s *K8SMetricsSurveyor) GetMetricsServerMetrics(nodes *v1.NodeList) ([]*ledger.MeasurementList, error) {
// 	podMetricsList, err := s.km.MetricsV1beta1().PodMetricses("").List(context.Background(), metav1.ListOptions{})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	timestamp := time.Now().UTC().Format(time.RFC3339)
//
// 	measurementLists := []*ledger.MeasurementList{}
//
// 	nodeTypes := make(map[string][]corev1.Node)
// 	for _, node := range nodes.Items {
// 		nodeType := node.Labels["beta.kubernetes.io/instance-type"]
// 		if _, exists := nodeTypes[nodeType]; exists {
// 			nodeTypes[nodeType] = append(nodeTypes[nodeType], node)
// 			continue
// 		}
// 		nodeTypes[nodeType] = []corev1.Node{node}
// 	}
//
// 	for _, nodes := range nodeTypes {
// 		sharedLabels := make(map[string]string)
// 		for k, v := range nodes[0].Labels {
// 			shared := true
// 			for _, nn := range nodes {
// 				if vv, ok := nn.Labels[k]; !ok || vv != v {
// 					shared = false
// 					break
// 				}
// 			}
// 			if shared {
// 				sharedLabels[k] = v
// 			}
// 		}
// 		nodeCountMeasurements := &ledger.MeasurementList{
// 			MeterName: "k8s_node_count",
// 			Labels:    sharedLabels,
// 			Measurements: []*ledger.Measurement{
// 				&ledger.Measurement{
// 					Value: float64(len(nodes)),
// 					Time:  timestamp,
// 				},
// 			},
// 		}
// 		measurementLists = append(measurementLists, nodeCountMeasurements)
// 	}
//
// 	podCountMeasurements := &ledger.MeasurementList{
// 		MeterName: "k8s_pod_count",
// 		Measurements: []*ledger.Measurement{
// 			&ledger.Measurement{
// 				Value: float64(len(podMetricsList.Items)),
// 				Time:  timestamp,
// 			},
// 		},
// 	}
// 	measurementLists = append(measurementLists, podCountMeasurements)
//
// 	for _, pm := range podMetricsList.Items {
// 		for _, container := range pm.Containers {
//
// 			cpuMillicores := float64(container.Usage.Cpu().ScaledValue(resource.Milli))
// 			memoryBytes := float64(container.Usage.Memory().ScaledValue(0))
//
// 			labels := make(map[string]string)
// 			if pm.Labels != nil {
// 				for k, v := range pm.Labels {
// 					labels[k] = v
// 				}
// 			}
// 			labels["container_name"] = container.Name
//
// 			cpuMeasurementList := &ledger.MeasurementList{
// 				MeterName: "k8s_cpu_milli",
// 				Namespace: pm.Namespace,
// 				Pod:       pm.Name,
// 				Labels:    labels,
// 				Measurements: []*ledger.Measurement{
// 					&ledger.Measurement{
// 						Value: cpuMillicores,
// 						Time:  timestamp,
// 					},
// 				},
// 			}
//
// 			memMeasurementList := &ledger.MeasurementList{
// 				MeterName: "k8s_mem_bytes",
// 				Namespace: pm.Namespace,
// 				Pod:       pm.Name,
// 				Labels:    labels,
// 				Measurements: []*ledger.Measurement{
// 					&ledger.Measurement{
// 						Value: memoryBytes,
// 						Time:  timestamp,
// 					},
// 				},
// 			}
//
// 			measurementLists = append(measurementLists, cpuMeasurementList)
// 			measurementLists = append(measurementLists, memMeasurementList)
// 		}
// 	}
//
// 	return measurementLists, nil
// }

func (s *K8SMetricsSurveyor) GetKubeStateMetrics() ([]*ledger.MeasurementList, error) {
	parsed, err := util.PrometheusExporterRequest(s.ksm)
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
					MeterName: metricName,
					Namespace: namespace,
					Pod:       pod,
					Labels:    labels,
					Measurements: []*ledger.Measurement{
						&ledger.Measurement{
							Value: value,
							Time:  timestamp,
						},
					},
				}
				measurementLists = append(measurementLists, measurementList)
			}
		}
	}

	return measurementLists, nil
}

func (s *K8SMetricsSurveyor) Survey(nodes *v1.NodeList) ([]*ledger.MeasurementList, error) {
	// metricsServerResult, err := s.GetMetricsServerMetrics(nodes)
	// if err != nil {
	// 	return nil, err
	// }

	kubeStateMetricsResult, err := s.GetKubeStateMetrics()
	if err != nil {
		return nil, err
	}

	// result := append(metricsServerResult, kubeStateMetricsResult...)
	result := kubeStateMetricsResult
	return result, nil
}
