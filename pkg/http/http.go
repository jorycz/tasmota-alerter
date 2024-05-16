package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	headers []string
)

// PUBLIC

func SetHeader(header string) {
	headers = append(headers, strings.TrimSpace(header))
}

func CallUrlWithOptionalTextData(url string, textData string) (statusCode int, body *bytes.Buffer) {
	var payload []byte
	var method = "GET"
	if len(textData) > 0 {
		payload = []byte(textData)
		method = "POST"
	}
	response, err := sendRequest(method, url, payload)
	if err != nil {
		slog.Error("Error sending request: %v\n", err)
		return 0, nil
	}
	responseBody := new(bytes.Buffer)
	_, err = responseBody.ReadFrom(response.Body)
	if err != nil {
		slog.Error("Error reading response: %v\n", err)
		return 0, nil
	}
	return response.StatusCode, responseBody
}

func CallUrlForDelete(url string) (statusCode int, body *bytes.Buffer) {
	var payload []byte
	var method = "DELETE"

	response, err := sendRequest(method, url, payload)
	if err != nil {
		slog.Error("Error sending request: %v\n", err)
		return 0, nil
	}
	responseBody := new(bytes.Buffer)
	_, err = responseBody.ReadFrom(response.Body)
	if err != nil {
		slog.Error("Error reading response: %v\n", err)
		return 0, nil
	}
	return response.StatusCode, responseBody
}

// PRIVATE

func sendRequest(method, urlStr string, payload []byte) (*http.Response, error) {
	var timeout = 5 * time.Second
	transport := &http.Transport{
		// Here goes proxy setup if any ...
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: transport,
	}

	slog.Debug("HTTP", "URL", urlStr, "timeout", timeout)

	var req *http.Request
	var err error

	if payload != nil {
		req, err = http.NewRequest(method, urlStr, bytes.NewBuffer(payload))
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequest(method, urlStr, nil)
		if err != nil {
			return nil, err
		}
	}

	for _, header := range headers {
		slog.Debug("HTTP", "Header", header)
		headerParts := strings.Split(header, ":")
		req.Header.Set(headerParts[0], headerParts[1])
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	
	// Delete all headers
	headers = []string{}
	return resp, nil
}
