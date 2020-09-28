package main

import (
	"github.com/getoctane/octane-collector/ledger"
)

func main() {
	lc := &ledger.Client{
		Scheme:     ledgerScheme,
		Host:       ledgerHost,
		ClusterKey: clusterKey,
	}

	q, err := newLedgerPushQueue()
	if err != nil {
		panic(err)
	}

	go startProxy(q)

	if enableSurveyors {
		go startSurveying(lc)
	}

	startMetering(lc)
}
