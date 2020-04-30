package surveyors

import (
	"context"
	"time"

	"github.com/cloudptio/octane/collector/ledger"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/clientcmd"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K8SMetricsSurveyor struct {
	k  *kubernetes.Clientset
	km *metricsv.Clientset
}

func NewK8SMetricsSurveyor(kubeconfig string) (*K8SMetricsSurveyor, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	km, err := metricsv.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &K8SMetricsSurveyor{k, km}, nil
}

func (s *K8SMetricsSurveyor) Survey() ([]*ledger.MeasurementList, error) {
	podMetricsList, err := s.km.MetricsV1beta1().PodMetricses("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMetricsList, err := s.km.MetricsV1beta1().NodeMetricses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodes, err := s.k.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	measurementLists := []*ledger.MeasurementList{}

	nodeTypes := make(map[string][]corev1.Node)
	for _, node := range nodes.Items {
		nodeType := node.Labels["beta.kubernetes.io/node-type"]
		if _, exists := nodeTypes[nodeType]; exists {
			nodeTypes[nodeType] = append(nodeTypes[nodeType], node)
			continue
		}
		nodeTypes[nodeType] = []corev1.Node{node}
	}

	for _, nodes := range nodeTypes {
		sharedLabels := make(map[string]string)
		for k, v := range nodes[0].Labels {
			shared := true
			for _, nn := range nodes {
				if vv, ok := nn.Labels[k]; !ok || vv != v {
					shared = false
					break
				}
			}
			if shared {
				sharedLabels[k] = v
			}
		}
		nodeCountMeasurements := &ledger.MeasurementList{
			MeterName: "k8s_node_count",
			Labels:    sharedLabels,
			Measurements: []*ledger.Measurement{
				&ledger.Measurement{
					Value: float64(len(nodeMetricsList.Items)),
					Time:  timestamp,
				},
			},
		}
		measurementLists = append(measurementLists, nodeCountMeasurements)
	}

	podCountMeasurements := &ledger.MeasurementList{
		MeterName: "k8s_pod_count",
		Measurements: []*ledger.Measurement{
			&ledger.Measurement{
				Value: float64(len(podMetricsList.Items)),
				Time:  timestamp,
			},
		},
	}
	measurementLists = append(measurementLists, podCountMeasurements)

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

			cpuMeasurementList := &ledger.MeasurementList{
				MeterName: "k8s_cpu_milli",
				Namespace: pm.Namespace,
				Pod:       pm.Name,
				Labels:    labels,
				Measurements: []*ledger.Measurement{
					&ledger.Measurement{
						Value: cpuMillicores,
						Time:  timestamp,
					},
				},
			}

			memMeasurementList := &ledger.MeasurementList{
				MeterName: "k8s_mem_bytes",
				Namespace: pm.Namespace,
				Pod:       pm.Name,
				Labels:    labels,
				Measurements: []*ledger.Measurement{
					&ledger.Measurement{
						Value: memoryBytes,
						Time:  timestamp,
					},
				},
			}

			measurementLists = append(measurementLists, cpuMeasurementList)
			measurementLists = append(measurementLists, memMeasurementList)
		}
	}

	return measurementLists, nil
}
