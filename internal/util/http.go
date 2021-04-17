package util

import (
	"bytes"
	//"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var transport = &http.Transport{
	//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

type RequestSetup func(*http.Request, *http.Client)

// Returns: status code, headers, body, error
func HttpRequest(method, url string, body []byte, timeoutSeconds int, settings ...RequestSetup) (int, http.Header, []byte, error) {
	var req *http.Request
	var err error
	if len(body) == 0 {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	}

	if err != nil {
		return 0, nil, nil, err
	}

	c := &http.Client{
		Transport: transport,
		Timeout:   time.Second * time.Duration(timeoutSeconds),
		//CheckRedirect: func(req *http.Request, via []*http.Request) error {
		//	return http.ErrUseLastResponse
		//},
	}
	for _, setup := range settings {
		setup(req, c)
	}
	res, err := c.Do(req)

	if err != nil {
		return 0, nil, nil, err
	}
	//defer res.Body.Close() // will do it in readAll

	data, err := readAll(res.Body)
	if err != nil {
		return res.StatusCode, res.Header, nil, err
	}

	return res.StatusCode, res.Header, data, nil
}

func readAll(r io.ReadCloser) ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	defer r.Close()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return data, nil
}
