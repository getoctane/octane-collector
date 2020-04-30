package meter_query

type Querier interface {
	GetUnitsConsumedForPeriod(query string, timespanSecs int64) (float64, error)
}
