package surveyors

import "github.com/getoctane/octane-collector/ledger"

type Surveyor interface {
	Survey() ([]*ledger.MeasurementList, error)
}
