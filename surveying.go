package main

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/surveyors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func kubeCfgAndClient() (*rest.Config, *kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, k, err
}

func startSurveying(lc *ledger.Client) {
	cfg, k, err := kubeCfgAndClient()
	if err != nil {
		panic(err)
	}

	allS := []surveyors.Surveyor{}

	metricsSurveyor, err := surveyors.NewMetricsServerSurveyor(cfg)
	if err != nil {
		panic(err)
	}

	kubeStateSurveyor, err := surveyors.NewKubeStateMetricsSurveyor(kubeStateMetricsHost)
	if err != nil {
		panic(err)
	}

	netcSurveyor, err := surveyors.NewKubeNetcSurveyor(k, kubeNetcNamespaceOverride, kubeNetcHostsOverride)
	if err != nil {
		panic(err)
	}

	allS = append(allS, metricsSurveyor)
	allS = append(allS, kubeStateSurveyor)
	allS = append(allS, netcSurveyor)

	for {
		// Sleep first so we give kube-state-metrics a chance to start
		time.Sleep(time.Duration(surveyingIntervalMinutes) * time.Minute)

		measurementLists := aggregateSurveys(allS)

		for _, measurements := range measurementLists {
			if err := lc.PostMeasurementList(measurements); err != nil {
				fmt.Printf("ERROR Failed to post measurement list: %s\n", err.Error())
			}
		}
	}
}

func aggregateSurveys(allS []surveyors.Surveyor) []*ledger.MeasurementList {
	allMeasurementLists := []*ledger.MeasurementList{}
	for _, s := range allS {
		measurementLists, err := s.Survey()
		if err != nil {
			fmt.Printf("ERROR Failed surveying: %s\n", err.Error())
			continue
		}
		allMeasurementLists = append(allMeasurementLists, measurementLists...)
	}

	idMap := make(map[[32]byte]*ledger.MeasurementList)

	for _, ml := range allMeasurementLists {
		id := createIdentifierForEntity(ml.Namespace, ml.Pod, ml.Labels)

		if _, exists := idMap[id]; !exists {
			idMap[id] = &ledger.MeasurementList{
				Namespace:    ml.Namespace,
				Pod:          ml.Pod,
				Labels:       ml.Labels,
				Measurements: []*ledger.Measurement{},
			}
		}
		idMap[id].Measurements = append(idMap[id].Measurements, ml.Measurements...)
	}

	aggregatedMeasurementLists := []*ledger.MeasurementList{}
	for _, mls := range idMap {
		aggregatedMeasurementLists = append(aggregatedMeasurementLists, mls)
	}

	return aggregatedMeasurementLists
}

func sortedAndStringifiedLabels(labels map[string]string) string {
	keys := make([]string, len(labels))
	i := 0
	for k := range labels {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	str := ""
	for _, k := range keys {
		str += k
		str += labels[k]
	}
	return str
}

func createIdentifierForEntity(namespace string, pod string, labels map[string]string) [32]byte {
	str := namespace + pod + sortedAndStringifiedLabels(labels)
	return sha256.Sum256([]byte(str))
}

func survey(lc *ledger.Client, s surveyors.Surveyor) error {
	measurementLists, err := s.Survey()
	if err != nil {
		return fmt.Errorf("ERROR Failed surveying: %s\n", err.Error())
	}

	for _, measurements := range measurementLists {
		if err := lc.PostMeasurementList(measurements); err != nil {
			fmt.Printf("ERROR Failed to post measurement list: %s\n", err.Error())
		}
	}

	return nil
}
