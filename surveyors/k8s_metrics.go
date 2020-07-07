package surveyors

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cloudptio/octane/collector/ledger"
	"github.com/prometheus/common/expfmt"
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
	k   *kubernetes.Clientset
	km  *metricsv.Clientset
	ksm string
}

func NewK8SMetricsSurveyor(kubeconfig string, ksm string) (*K8SMetricsSurveyor, error) {
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
	return &K8SMetricsSurveyor{k, km, ksm}, nil
}

func (s *K8SMetricsSurveyor) GetMetricsServerMetrics() ([]*ledger.MeasurementList, error) {
	podMetricsList, err := s.km.MetricsV1beta1().PodMetricses("").List(context.Background(), metav1.ListOptions{})
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
		nodeType := node.Labels["beta.kubernetes.io/instance-type"]
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
					Value: float64(len(nodes)),
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

func (s *K8SMetricsSurveyor) GetKubeStateMetrics() ([]*ledger.MeasurementList, error) {
	url := fmt.Sprintf("%s/metrics", s.ksm)
	timestamp := time.Now().UTC().Format(time.RFC3339)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for url %s: %s", url, err.Error())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for url %s: %s", url, err.Error())
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of response: %s", err.Error())
	}

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus text: %s", err.Error())
	}

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
			measurmentList := &ledger.MeasurementList{
				MeterName: metricName,
				Namespace: namespace,
				Pod:       pod,
				Labels:    labels,
				Measurements: []*ledger.Measurement{
					&ledger.Measurement{
						Value: m.GetGauge().GetValue(),
						Time:  timestamp,
					},
				},
			}
			measurementLists = append(measurementLists, measurmentList)
		}
	}

	return measurementLists, nil
}

func (s *K8SMetricsSurveyor) Survey() ([]*ledger.MeasurementList, error) {
	metricsServerResult, err := s.GetMetricsServerMetrics()
	if err != nil {
		return nil, err
	}

	kubeStateMetricsResult, err := s.GetKubeStateMetrics()
	if err != nil {
		return nil, err
	}

	result := append(metricsServerResult, kubeStateMetricsResult...)
	return result, nil
}
