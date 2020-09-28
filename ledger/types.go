package ledger

import "strconv"

type Meter struct {
	ID              int64  `json:"id"`
	Type            string `json:"meter_type"`
	Value           string `json:"value"`
	TimespanMinutes int64  `json:"timespan_minutes"`
}

type MeterList struct {
	Meters []*Meter `json:"meters"`
}

func (m *Meter) Name() string {
	return strconv.FormatInt(m.ID, 10)
}

type Measurement struct {
	MeterID   int64   `json:"meter_id,omitempty"`
	MeterName string  `json:"meter_name,omitempty"`
	Value     float64 `json:"value"`
	Time      string  `json:"time"`
}

type MeasurementList struct {
	Namespace    string            `json:"namespace,omitempty"`
	Pod          string            `json:"pod,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Measurements []*Measurement    `json:"measurements,omitempty"`
}
