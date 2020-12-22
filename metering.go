package main

import (
	"fmt"
	"time"

	"github.com/getoctane/octane-collector/ledger"
)

type meterer struct {
	lc *ledger.Client
}

func startMetering(lc *ledger.Client) {
	m := &meterer{lc}
	for {
		m.meter()
		time.Sleep(time.Duration(meteringIntervalMinutes) * time.Minute)
	}
}

func (m *meterer) meter() {
	meters, err := m.lc.ListMeters()
	if err != nil {
		fmt.Printf("ERROR Failed to list meters: %s\n", err.Error())
		return
	}

	for _, meter := range meters {
		timespanSecs := int64(60)
		timestamp := time.Now().UTC().Format(time.RFC3339)

		querier, err := querierFor(meter)
		if err != nil {
			fmt.Printf("ERROR Failed to get Querier on Meter %s: %s\n", meter.Name(), err.Error())
			continue
		}

		// For custom meters
		if querier == nil {
			continue
		}

		units, err := querier.GetUnitsConsumedForPeriod(meter.Query, timespanSecs)
		if err != nil {
			fmt.Printf("ERROR Failed to get meter units on Meter %s: %s\n", meter.Name(), err.Error())
			continue
		}

		if err := m.lc.CreateMeasurement(meter, timestamp, units); err != nil {
			fmt.Printf("ERROR Failed to send measurement on Meter %s: %s\n", meter.Name(), err.Error())
			continue
		}
	}
}
