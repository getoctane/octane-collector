package surveyors

import "github.com/cloudptio/octane/collector/ledger"

type Surveyor interface {
	Survey() ([]*ledger.MeasurementList, error)
}
