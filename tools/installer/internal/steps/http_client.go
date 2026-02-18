package steps

import (
	"crypto/tls"
	"net/http"
	"time"
)

func newHTTPClient(tlsConfig *tls.Config, timeout time.Duration) *http.Client {
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	if tlsConfig != nil {
		config = tlsConfig.Clone()
		if config.MinVersion == 0 {
			config.MinVersion = tls.VersionTLS12
		}
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
	}
}

func checkURLReady(client *http.Client, targetURL string) bool {
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return false
	}

	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 400
}

func checkURLWarm(client *http.Client, targetURL string) bool {
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return false
	}

	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode >= 200 && response.StatusCode < 500
}
