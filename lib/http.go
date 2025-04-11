package lib

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"
)

func HttpAddUrlQuery(rawUrl string, urlQuery map[string]string) (string, error) {
	urlParse, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	urlValues := urlParse.Query()
	for key, value := range urlQuery {
		urlValues.Add(key, value)
	}
	urlParse.RawQuery = urlValues.Encode()
	return urlParse.String(), nil
}

func HttpRequest(method string, rawUrl string, body []byte, clientRequest func(client *http.Client, request *http.Request) error, handler func(request *http.Request, response *http.Response) error) error {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}
	request, err := http.NewRequest(method, rawUrl, bodyReader)
	if err != nil {
		return err
	}
	client := &http.Client{}
	client.Timeout = time.Second * 5
	if clientRequest != nil {
		if err = clientRequest(client, request); err != nil {
			return err
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	return handler(request, response)
}

func HttpGet(rawUrl string, clientRequest func(client *http.Client, request *http.Request) error, handler func(request *http.Request, response *http.Response) error) error {
	return HttpRequest(http.MethodGet, rawUrl, nil, clientRequest, handler)
}

func HttpPost(rawUrl string, body []byte, clientRequest func(client *http.Client, request *http.Request) error, handler func(request *http.Request, response *http.Response) error) error {
	return HttpRequest(http.MethodPost, rawUrl, body, clientRequest, handler)
}

func HttpPut(rawUrl string, body []byte, clientRequest func(client *http.Client, request *http.Request) error, handler func(request *http.Request, response *http.Response) error) error {
	return HttpRequest(http.MethodPut, rawUrl, body, clientRequest, handler)
}

func HttpDelete(rawUrl string, body []byte, clientRequest func(client *http.Client, request *http.Request) error, handler func(request *http.Request, response *http.Response) error) error {
	return HttpRequest(http.MethodDelete, rawUrl, body, clientRequest, handler)
}
