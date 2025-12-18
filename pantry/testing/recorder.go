// testing/recorder.go
package testing

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Recorder provides a way to test handlers without starting a server.
type Recorder struct {
	t *testing.T
}

// NewRecorder creates a new Recorder for testing.
func NewRecorder(t *testing.T) *Recorder {
	return &Recorder{t: t}
}

// Request creates a new recorder request builder.
func (rec *Recorder) Request(method, path string) *RecorderRequest {
	return &RecorderRequest{
		rec:    rec,
		method: method,
		path:   path,
		header: make(http.Header),
		query:  make(url.Values),
	}
}

// Get creates a GET request.
func (rec *Recorder) Get(path string) *RecorderRequest {
	return rec.Request(http.MethodGet, path)
}

// Post creates a POST request.
func (rec *Recorder) Post(path string) *RecorderRequest {
	return rec.Request(http.MethodPost, path)
}

// Put creates a PUT request.
func (rec *Recorder) Put(path string) *RecorderRequest {
	return rec.Request(http.MethodPut, path)
}

// Patch creates a PATCH request.
func (rec *Recorder) Patch(path string) *RecorderRequest {
	return rec.Request(http.MethodPatch, path)
}

// Delete creates a DELETE request.
func (rec *Recorder) Delete(path string) *RecorderRequest {
	return rec.Request(http.MethodDelete, path)
}

// RecorderRequest builds a request for handler testing.
type RecorderRequest struct {
	rec    *Recorder
	method string
	path   string
	header http.Header
	query  url.Values
	body   io.Reader
}

// Header sets a request header.
func (rr *RecorderRequest) Header(key, value string) *RecorderRequest {
	rr.header.Set(key, value)
	return rr
}

// Headers sets multiple request headers.
func (rr *RecorderRequest) Headers(headers map[string]string) *RecorderRequest {
	for k, v := range headers {
		rr.header.Set(k, v)
	}
	return rr
}

// Query sets a query parameter.
func (rr *RecorderRequest) Query(key, value string) *RecorderRequest {
	rr.query.Set(key, value)
	return rr
}

// Queries sets multiple query parameters.
func (rr *RecorderRequest) Queries(params map[string]string) *RecorderRequest {
	for k, v := range params {
		rr.query.Set(k, v)
	}
	return rr
}

// Body sets the request body.
func (rr *RecorderRequest) Body(body io.Reader) *RecorderRequest {
	rr.body = body
	return rr
}

// BodyString sets the request body from a string.
func (rr *RecorderRequest) BodyString(body string) *RecorderRequest {
	rr.body = strings.NewReader(body)
	return rr
}

// BodyBytes sets the request body from bytes.
func (rr *RecorderRequest) BodyBytes(body []byte) *RecorderRequest {
	rr.body = bytes.NewReader(body)
	return rr
}

// JSON sets the request body as JSON.
func (rr *RecorderRequest) JSON(v any) *RecorderRequest {
	data, err := json.Marshal(v)
	if err != nil {
		rr.rec.t.Fatalf("failed to marshal JSON: %v", err)
	}
	rr.body = bytes.NewReader(data)
	rr.header.Set("Content-Type", "application/json")
	return rr
}

// Form sets the request body as form data.
func (rr *RecorderRequest) Form(data url.Values) *RecorderRequest {
	rr.body = strings.NewReader(data.Encode())
	rr.header.Set("Content-Type", "application/x-www-form-urlencoded")
	return rr
}

// Bearer sets the Authorization header with a Bearer token.
func (rr *RecorderRequest) Bearer(token string) *RecorderRequest {
	rr.header.Set("Authorization", "Bearer "+token)
	return rr
}

// BasicAuth sets Basic auth header.
func (rr *RecorderRequest) BasicAuth(username, password string) *RecorderRequest {
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth(username, password)
	rr.header.Set("Authorization", req.Header.Get("Authorization"))
	return rr
}

// Cookie adds a cookie to the request.
func (rr *RecorderRequest) Cookie(name, value string) *RecorderRequest {
	existing := rr.header.Get("Cookie")
	if existing != "" {
		rr.header.Set("Cookie", existing+"; "+name+"="+value)
	} else {
		rr.header.Set("Cookie", name+"="+value)
	}
	return rr
}

// Build creates the http.Request without executing it.
func (rr *RecorderRequest) Build() *http.Request {
	rr.rec.t.Helper()

	path := rr.path
	if len(rr.query) > 0 {
		path += "?" + rr.query.Encode()
	}

	req := httptest.NewRequest(rr.method, path, rr.body)
	req.Header = rr.header

	return req
}

// Run executes the request against a handler and returns the response.
func (rr *RecorderRequest) Run(handler http.Handler) *Response {
	rr.rec.t.Helper()

	req := rr.Build()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		rr.rec.t.Fatalf("failed to read response body: %v", err)
	}

	return &Response{
		Response: resp,
		Body:     body,
		t:        rr.rec.t,
	}
}

// RunFunc executes the request against a handler function.
func (rr *RecorderRequest) RunFunc(handler http.HandlerFunc) *Response {
	return rr.Run(handler)
}
