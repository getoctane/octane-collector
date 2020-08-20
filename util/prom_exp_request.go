package util

import (
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func PrometheusExporterRequest(host string) (map[string]*dto.MetricFamily, error) {
	bodyBytes, err := HttpRequest("GET", host+"/metrics", nil, nil)
	if err != nil {
		return nil, err
	}

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus text: %s", err.Error())
	}

	return parsed, nil
}
