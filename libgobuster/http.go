package libgobuster

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	userAgent = "gobuster"
)

type httpClient struct {
	client        *http.Client
	context       context.Context
	userAgent     string
	username      string
	password      string
	includeLength bool
}

// NewHTTPClient returns a new HTTPClient
func newHTTPClient(c context.Context, opt *Options) (*httpClient, error) {
	var proxyURLFunc func(*http.Request) (*url.URL, error)
	var client httpClient
	proxyURLFunc = http.ProxyFromEnvironment

	if opt.Proxy != "" {
		proxyURL, err := url.Parse(opt.Proxy)
		if err != nil {
			return nil, fmt.Errorf("[!] Proxy URL is invalid")
		}
		proxyURLFunc = http.ProxyURL(proxyURL)
	}

	var redirectFunc func(req *http.Request, via []*http.Request) error
	if !opt.FollowRedirect {
		redirectFunc = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		redirectFunc = nil
	}

	client.client = &http.Client{
		Timeout:       opt.Timeout,
		CheckRedirect: redirectFunc,
		Transport: &http.Transport{
			Proxy: proxyURLFunc,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opt.InsecureSSL,
			},
		}}
	client.context = c
	client.username = opt.Username
	client.password = opt.Password
	client.includeLength = opt.IncludeLength
	return &client, nil
}

// MakeRequest makes a request to the specified url
func (client *httpClient) makeRequest(fullURL, cookie string) (*int, *int64, error) {
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)

	if err != nil {
		return nil, nil, err
	}

	// add the context so we can easily cancel out
	req = req.WithContext(client.context)

	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	ua := userAgent
	if client.userAgent != "" {
		ua = client.userAgent
	}
	req.Header.Set("User-Agent", ua)

	if client.username != "" {
		req.SetBasicAuth(client.username, client.password)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		if ue, ok := err.(*url.Error); ok {

			if strings.HasPrefix(ue.Err.Error(), "x509") {
				return nil, nil, fmt.Errorf("invalid certificate")
			}
		}
		return nil, nil, err
	}

	defer resp.Body.Close()

	var length *int64

	if client.includeLength {
		length = new(int64)
		if resp.ContentLength <= 0 {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				*length = int64(utf8.RuneCountInString(string(body)))
			}
		} else {
			*length = resp.ContentLength
		}
	}

	return &resp.StatusCode, length, nil
}