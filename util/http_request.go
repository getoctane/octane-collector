package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type ErrorHTTP struct {
	code int
	body string
}

func (err *ErrorHTTP) Error() string {
	return fmt.Sprintf("HTTP error %d: %s", err.code, err.body)
}

func HttpRequest(method string, url string, headers http.Header, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for url %s: %s", url, err.Error())
	}

	if headers != nil {
		req.Header = headers
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for url %s: %s", url, err.Error())
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of response: %s", err.Error())
	}

	switch resp.StatusCode {
	case 401, 404:
		return nil, &ErrorHTTP{resp.StatusCode, string(bodyBytes)}
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Status %d from %s: %s", resp.StatusCode, url, string(bodyBytes))
	}

	return bodyBytes, nil
}
