package meter_query

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

// PrometheusMeterQuery implements the Querier interface.
type PrometheusMeterQuery struct {
	prometheusHost string
}

func NewPrometheusMeterQuery(host string) (*PrometheusMeterQuery, error) {
	if host == "" {
		return nil, errors.New("PROMETHEUS_HOST not defined! Meters of type Prometheus will not function.")
	}
	return &PrometheusMeterQuery{host}, nil
}

type prometheusResponse struct {
	Data prometheusResponseData `json:"data"`
}

type prometheusResponseData struct {
	Result []prometheusResult `json:"result"`
}

type prometheusResult struct {
	Value [2]interface{} `json:"value"`
}

func (pmq *PrometheusMeterQuery) GetUnitsConsumedForPeriod(query string, timespanSecs int64) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/query", pmq.prometheusHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0.0, err
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0.0, err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0.0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0.0, fmt.Errorf("HTTP status %d from Prometheus on %s: %s", resp.StatusCode, url, bodyBytes)
	}

	var dst prometheusResponse
	if err := json.Unmarshal(bodyBytes, &dst); err != nil {
		return 0.0, err
	}

	if len(dst.Data.Result) == 0 {
		return 0.0, nil
	}

	valueString := dst.Data.Result[0].Value[1].(string)

	return strconv.ParseFloat(valueString, 64)
}
