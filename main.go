package main

import "github.com/cloudptio/octane/collector/ledger"

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

	if enableK8SMetricsSurveyor {
		go startSurveying(lc)
	}

	startMetering(lc)
}
