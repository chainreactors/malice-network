package httputils

import (
	"io"
	"net/http"
	"time"

	"bytes"
	"encoding/json"
	"fmt"
)

var HttpClient = &http.Client{
	Timeout: 60 * time.Second,
}

func DoRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return HttpClient.Do(req)
}

func DoJSONRequest(method, url string, body io.Reader, headers map[string]string, expectStatus int, out interface{}) error {
	resp, err := DoRequest(method, url, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectStatus {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func DoGET(url string, headers map[string]string, out interface{}) error {
	return DoJSONRequest("GET", url, nil, headers, 200, out)
}

func DoPOST(url string, data interface{}, headers map[string]string, expectStatus int, out interface{}) error {
	var body io.Reader
	if data != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(jsonBytes)
		headers["Content-Type"] = "application/json"
	}
	return DoJSONRequest("POST", url, body, headers, expectStatus, out)
}
