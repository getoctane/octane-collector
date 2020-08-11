package main

import (
	"fmt"

	"github.com/getoctane/octane-collector/ledger"
	"github.com/getoctane/octane-collector/meter_query"
)

func querierFor(m *ledger.Meter) (meter_query.Querier, error) {
	switch m.Type {

	case "prometheus":
		return meter_query.NewPrometheusMeterQuery(prometheusHost)

	case "custom":
		return nil, nil

	default:
		return nil, fmt.Errorf("Don't know Meter type '%s'", m.Type)
	}
}
