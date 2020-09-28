package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

type Client struct {
	Scheme     string
	Host       string
	ClusterKey string
}

func (c *Client) ListMeters() ([]*Meter, error) {
	// dst := &MeterList{}
	dst := []*Meter{}

	if err := c.makeRequest("GET", "/instance/meters", nil, &dst); err != nil {
		return nil, err
	}

	// return dst.Meters, nil
	return dst, nil
}

func (c *Client) CreateMeasurement(meter *Meter, timestamp string, units float64) error {
	measurements := &MeasurementList{
		Measurements: []*Measurement{
			&Measurement{
				Value:   units,
				Time:    timestamp,
				MeterID: meter.ID,
			},
		},
	}
	return c.PostMeasurementList(measurements)
}

// This uses the proxy, which utilizes the queue
func (c *Client) PostMeasurementList(measurements *MeasurementList) error {
	return c.makeRequestWithHost("http", "localhost:8081", "POST", "/instance/measurements", measurements, nil)
}

func (c *Client) makeRequest(method string, path string, reqBodyObj interface{}, respBodyObj interface{}) error {
	return c.makeRequestWithHost(c.Scheme, c.Host, method, path, reqBodyObj, respBodyObj)
}

func (c *Client) makeRequestWithHost(scheme string, host string, method string, path string, reqBodyObj interface{}, respBodyObj interface{}) error {
	url := fmt.Sprintf("%s://%s", scheme, filepath.Join(host, path))

	var buf io.ReadWriter
	if reqBodyObj != nil {
		buf = new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(reqBodyObj); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.ClusterKey)

	if buf != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d from %s %s: %s", resp.StatusCode, method, url, string(bodyBytes))
	}

	if respBodyObj != nil {
		if err := json.Unmarshal(bodyBytes, respBodyObj); err != nil {
			return err
		}
	}

	return nil
}
