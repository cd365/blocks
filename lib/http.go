package lib

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

func HttpAddUrlQuery(urlString string, urlQuery map[string]string) (string, error) {
	urlParse, err := url.Parse(urlString)
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

// HttpRequestUserAgent Customize the User-Agent header of the http request.
func HttpRequestUserAgent(request *http.Request, userAgent string) {
	if userAgent == "" {
		userAgent = "HTTP Client"
	}
	request.Header.Set("User-Agent", userAgent)
}

// HttpRequestResponse Dynamically construct request data and dynamically process response results.
type HttpRequestResponse struct {
	RequestMethod string       // request method
	RequestUrl    string       // request url
	RequestBody   []byte       // request body
	Client        *http.Client // request client

	Request      *http.Request  // http request
	Response     *http.Response // http response
	responseBody []byte         // http response body
	responseRead int32          // ensure Response.Body is only read once

	handleRequest  func(rr *HttpRequestResponse) error // dynamically construct request data
	handleResponse func(rr *HttpRequestResponse) error // dynamically process response results
}

func (s *HttpRequestResponse) AddHandleRequest(handle func(rr *HttpRequestResponse) error) *HttpRequestResponse {
	if handle == nil {
		return s
	}
	tmp := s.handleRequest
	if tmp == nil {
		s.handleRequest = handle
	} else {
		s.handleRequest = func(rr *HttpRequestResponse) error {
			if err := tmp(rr); err != nil {
				return err
			}
			if err := handle(rr); err != nil {
				return err
			}
			return nil
		}
	}
	return s
}

func (s *HttpRequestResponse) AddHandleResponse(handle func(rr *HttpRequestResponse) error) *HttpRequestResponse {
	if handle == nil {
		return s
	}
	tmp := s.handleResponse
	if tmp == nil {
		s.handleResponse = handle
	} else {
		s.handleResponse = func(rr *HttpRequestResponse) error {
			if err := tmp(rr); err != nil {
				return err
			}
			if err := handle(rr); err != nil {
				return err
			}
			return nil
		}
	}
	return s
}

func (s *HttpRequestResponse) GetHandleRequest() func(*HttpRequestResponse) error {
	return s.handleRequest
}

func (s *HttpRequestResponse) GetHandleResponse() func(*HttpRequestResponse) error {
	return s.handleResponse
}

func (s *HttpRequestResponse) SetHandleRequest(handle func(rr *HttpRequestResponse) error) *HttpRequestResponse {
	s.handleRequest = handle
	return s
}

func (s *HttpRequestResponse) SetHandleResponse(handle func(rr *HttpRequestResponse) error) *HttpRequestResponse {
	s.handleResponse = handle
	return s
}

func (s *HttpRequestResponse) GetResponseBody() ([]byte, error) {
	if s.Response == nil {
		return nil, errors.New("response is <nil>")
	}
	if atomic.CompareAndSwapInt32(&s.responseRead, 0, 1) {
		buffer := bytes.NewBuffer(nil)
		if _, err := io.Copy(buffer, s.Response.Body); err != nil {
			return nil, err
		}
		s.responseBody = buffer.Bytes()
	}
	return s.responseBody, nil
}

func HttpRequest(method string, urlString string, handle func(rr *HttpRequestResponse) error) error {
	client := &http.Client{}
	client.Timeout = time.Second * 5

	tmp := &HttpRequestResponse{
		RequestMethod: method,
		RequestUrl:    urlString,
		Client:        client,
	}

	if handle != nil {
		if err := handle(tmp); err != nil {
			return err
		}
	}

	reader := io.Reader(nil)
	if len(tmp.RequestBody) > 0 {
		reader = bytes.NewBuffer(tmp.RequestBody)
	}

	request, err := http.NewRequest(tmp.RequestMethod, tmp.RequestUrl, reader)
	if err != nil {
		return err
	}
	tmp.Request = request

	if fc := tmp.handleRequest; fc != nil {
		if err = fc(tmp); err != nil {
			return err
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	tmp.Response = response

	defer func() { _ = response.Body.Close() }()

	if fc := tmp.handleResponse; fc != nil {
		if err = fc(tmp); err != nil {
			return err
		}
	}

	return nil
}

func HttpGet(urlString string, handle func(rr *HttpRequestResponse) error) error {
	return HttpRequest(http.MethodGet, urlString, handle)
}

func HttpPost(urlString string, handle func(rr *HttpRequestResponse) error) error {
	return HttpRequest(http.MethodPost, urlString, handle)
}

func HttpPut(urlString string, handle func(rr *HttpRequestResponse) error) error {
	return HttpRequest(http.MethodPut, urlString, handle)
}

func HttpDelete(urlString string, handle func(rr *HttpRequestResponse) error) error {
	return HttpRequest(http.MethodDelete, urlString, handle)
}
