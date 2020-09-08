package surveyors

import (
	"github.com/getoctane/octane-collector/ledger"
	v1 "k8s.io/api/core/v1"
)

type Surveyor interface {
	Survey(*v1.NodeList) ([]*ledger.MeasurementList, error)
}
