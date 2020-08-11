package main

import (
	"fmt"
	"time"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/surveyors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// this determines how frequently to collect Measurements from Meters
	surveyingInterval = time.Minute
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

	metricsSurveyor, err := surveyors.NewK8SMetricsSurveyor(cfg, k, kubeStateMetricsHost)
	if err != nil {
		panic(err)
	}

	netcSurveyor, err := surveyors.NewKubeNetcSurveyor(k)
	if err != nil {
		panic(err)
	}

	allS = append(allS, metricsSurveyor)
	allS = append(allS, netcSurveyor)

	for {
		for _, s := range allS {
			if err := survey(lc, s); err != nil {
				fmt.Println(err)
			}
		}
		time.Sleep(surveyingInterval)
	}
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
