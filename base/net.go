package base

import (
	"net/http"
	"net/url"
	"time"
)

func NewHttpClient(proxy string, timeout time.Duration) *http.Client {
	var (
		err        error
		proxyURL   *url.URL
		httpClient *http.Client
	)
	if len(proxy) > 0 {
		proxyURL, err = url.Parse(proxy)
	}
	if err == nil && proxyURL != nil {
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		httpClient = &http.Client{Timeout: timeout, Transport: transport}
	} else {
		httpClient = &http.Client{Timeout: timeout}
	}
	return httpClient
}
