package httputils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var HttpClient = &http.Client{
	Timeout: 180 * time.Second,
}

func DoRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	return DoRequestContext(context.Background(), method, url, body, headers)
}

func DoRequestContext(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return HttpClient.Do(req)
}

func DoJSONRequest(method, url string, body io.Reader, headers map[string]string, expectStatus int, out interface{}) error {
	return DoJSONRequestContext(context.Background(), method, url, body, headers, expectStatus, out)
}

func DoJSONRequestContext(ctx context.Context, method, url string, body io.Reader, headers map[string]string, expectStatus int, out interface{}) error {
	resp, err := DoRequestContext(ctx, method, url, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectStatus {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}
	if out != nil {
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(payload)) == 0 {
			return nil
		}
		return json.Unmarshal(payload, out)
	}
	return nil
}

func DoGET(url string, headers map[string]string, out interface{}) error {
	return DoJSONRequest("GET", url, nil, headers, 200, out)
}

func DoPOST(url string, data interface{}, headers map[string]string, expectStatus int, out interface{}) error {
	return DoPOSTContext(context.Background(), url, data, headers, expectStatus, out)
}

func DoPOSTContext(ctx context.Context, url string, data interface{}, headers map[string]string, expectStatus int, out interface{}) error {
	var body io.Reader
	mergedHeaders := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		mergedHeaders[k] = v
	}
	if data != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(jsonBytes)
		mergedHeaders["Content-Type"] = "application/json"
	}
	return DoJSONRequestContext(ctx, "POST", url, body, mergedHeaders, expectStatus, out)
}
