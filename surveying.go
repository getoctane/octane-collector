package main

import (
	"fmt"
	"time"

	"github.com/cloudptio/octane/collector/ledger"
	"github.com/cloudptio/octane/collector/surveyors"
)

const (
	// this determines how frequently to collect Measurements from Meters
	surveyingInterval = time.Minute
)

func startSurveying(lc *ledger.Client) {
	s, err := surveyors.NewK8SMetricsSurveyor(kubeconfig)
	if err != nil {
		panic(err)
	}
	for {
		if err := survey(lc, s); err != nil {
			fmt.Println(err)
		}
		time.Sleep(surveyingInterval)
	}
}

func survey(lc *ledger.Client, s surveyors.Surveyor) error {
	measurementLists, err := s.Survey()
	if err != nil {
		return fmt.Errorf("ERROR Failed surveying K8S metrics: %s\n", err.Error())
	}

	for _, measurements := range measurementLists {
		if err := lc.PostMeasurementList(measurements); err != nil {
			fmt.Printf("ERROR Failed to post measurement list for K8S metrics: %s\n", err.Error())
		}
	}

	return nil
}
