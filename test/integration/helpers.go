//go:build integration
// +build integration

package integration

import (
	"fmt"
	"net/http"
	"time"
)

// waitForService waits for a service to become available
func waitForService(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}

		fmt.Printf("Waiting for service at %s...\n", url)
		time.Sleep(2 * time.Second)
	}

	return false
}

// makeRequest is a helper to make HTTP requests with optional headers
func makeRequest(method, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return http.DefaultClient.Do(req)
}
