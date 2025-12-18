// testing/testing.go
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

	"github.com/go-chi/chi/v5"
)

// Server wraps httptest.Server with convenience methods for testing.
type Server struct {
	*httptest.Server
	Router chi.Router
	t      *testing.T
}

// NewServer creates a test server with the given router.
func NewServer(t *testing.T, r chi.Router) *Server {
	t.Helper()
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &Server{
		Server: srv,
		Router: r,
		t:      t,
	}
}

// NewTLSServer creates a test server with TLS.
func NewTLSServer(t *testing.T, r chi.Router) *Server {
	t.Helper()
	srv := httptest.NewTLSServer(r)
	t.Cleanup(srv.Close)
	return &Server{
		Server: srv,
		Router: r,
		t:      t,
	}
}

// Request creates a new request builder for the server.
func (s *Server) Request(method, path string) *RequestBuilder {
	return &RequestBuilder{
		server: s,
		method: method,
		path:   path,
		header: make(http.Header),
		query:  make(url.Values),
	}
}

// Get creates a GET request builder.
func (s *Server) Get(path string) *RequestBuilder {
	return s.Request(http.MethodGet, path)
}

// Post creates a POST request builder.
func (s *Server) Post(path string) *RequestBuilder {
	return s.Request(http.MethodPost, path)
}

// Put creates a PUT request builder.
func (s *Server) Put(path string) *RequestBuilder {
	return s.Request(http.MethodPut, path)
}

// Patch creates a PATCH request builder.
func (s *Server) Patch(path string) *RequestBuilder {
	return s.Request(http.MethodPatch, path)
}

// Delete creates a DELETE request builder.
func (s *Server) Delete(path string) *RequestBuilder {
	return s.Request(http.MethodDelete, path)
}

// RequestBuilder builds and executes HTTP requests.
type RequestBuilder struct {
	server *Server
	method string
	path   string
	header http.Header
	query  url.Values
	body   io.Reader
}

// Header sets a request header.
func (rb *RequestBuilder) Header(key, value string) *RequestBuilder {
	rb.header.Set(key, value)
	return rb
}

// Headers sets multiple request headers.
func (rb *RequestBuilder) Headers(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		rb.header.Set(k, v)
	}
	return rb
}

// Query sets a query parameter.
func (rb *RequestBuilder) Query(key, value string) *RequestBuilder {
	rb.query.Set(key, value)
	return rb
}

// Queries sets multiple query parameters.
func (rb *RequestBuilder) Queries(params map[string]string) *RequestBuilder {
	for k, v := range params {
		rb.query.Set(k, v)
	}
	return rb
}

// Body sets the request body.
func (rb *RequestBuilder) Body(body io.Reader) *RequestBuilder {
	rb.body = body
	return rb
}

// BodyString sets the request body from a string.
func (rb *RequestBuilder) BodyString(body string) *RequestBuilder {
	rb.body = strings.NewReader(body)
	return rb
}

// BodyBytes sets the request body from bytes.
func (rb *RequestBuilder) BodyBytes(body []byte) *RequestBuilder {
	rb.body = bytes.NewReader(body)
	return rb
}

// JSON sets the request body as JSON and sets the Content-Type header.
func (rb *RequestBuilder) JSON(v any) *RequestBuilder {
	data, err := json.Marshal(v)
	if err != nil {
		rb.server.t.Fatalf("failed to marshal JSON: %v", err)
	}
	rb.body = bytes.NewReader(data)
	rb.header.Set("Content-Type", "application/json")
	return rb
}

// Form sets the request body as form data and sets the Content-Type header.
func (rb *RequestBuilder) Form(data url.Values) *RequestBuilder {
	rb.body = strings.NewReader(data.Encode())
	rb.header.Set("Content-Type", "application/x-www-form-urlencoded")
	return rb
}

// Bearer sets the Authorization header with a Bearer token.
func (rb *RequestBuilder) Bearer(token string) *RequestBuilder {
	rb.header.Set("Authorization", "Bearer "+token)
	return rb
}

// BasicAuth sets the Authorization header with Basic auth.
func (rb *RequestBuilder) BasicAuth(username, password string) *RequestBuilder {
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth(username, password)
	rb.header.Set("Authorization", req.Header.Get("Authorization"))
	return rb
}

// Cookie adds a cookie to the request.
func (rb *RequestBuilder) Cookie(name, value string) *RequestBuilder {
	existing := rb.header.Get("Cookie")
	if existing != "" {
		rb.header.Set("Cookie", existing+"; "+name+"="+value)
	} else {
		rb.header.Set("Cookie", name+"="+value)
	}
	return rb
}

// Do executes the request and returns a Response.
func (rb *RequestBuilder) Do() *Response {
	rb.server.t.Helper()

	// Build URL with query params
	u := rb.server.URL + rb.path
	if len(rb.query) > 0 {
		u += "?" + rb.query.Encode()
	}

	req, err := http.NewRequest(rb.method, u, rb.body)
	if err != nil {
		rb.server.t.Fatalf("failed to create request: %v", err)
	}

	req.Header = rb.header

	client := rb.server.Client()
	resp, err := client.Do(req)
	if err != nil {
		rb.server.t.Fatalf("failed to execute request: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		rb.server.t.Fatalf("failed to read response body: %v", err)
	}

	return &Response{
		Response: resp,
		Body:     body,
		t:        rb.server.t,
	}
}

// Response wraps http.Response with assertion methods.
type Response struct {
	*http.Response
	Body []byte
	t    *testing.T
}

// Status asserts the response status code.
func (r *Response) Status(code int) *Response {
	r.t.Helper()
	if r.StatusCode != code {
		r.t.Errorf("expected status %d, got %d\nBody: %s", code, r.StatusCode, string(r.Body))
	}
	return r
}

// StatusOK asserts 200 OK.
func (r *Response) StatusOK() *Response {
	return r.Status(http.StatusOK)
}

// StatusCreated asserts 201 Created.
func (r *Response) StatusCreated() *Response {
	return r.Status(http.StatusCreated)
}

// StatusNoContent asserts 204 No Content.
func (r *Response) StatusNoContent() *Response {
	return r.Status(http.StatusNoContent)
}

// StatusBadRequest asserts 400 Bad Request.
func (r *Response) StatusBadRequest() *Response {
	return r.Status(http.StatusBadRequest)
}

// StatusUnauthorized asserts 401 Unauthorized.
func (r *Response) StatusUnauthorized() *Response {
	return r.Status(http.StatusUnauthorized)
}

// StatusForbidden asserts 403 Forbidden.
func (r *Response) StatusForbidden() *Response {
	return r.Status(http.StatusForbidden)
}

// StatusNotFound asserts 404 Not Found.
func (r *Response) StatusNotFound() *Response {
	return r.Status(http.StatusNotFound)
}

// StatusConflict asserts 409 Conflict.
func (r *Response) StatusConflict() *Response {
	return r.Status(http.StatusConflict)
}

// StatusUnprocessableEntity asserts 422 Unprocessable Entity.
func (r *Response) StatusUnprocessableEntity() *Response {
	return r.Status(http.StatusUnprocessableEntity)
}

// StatusTooManyRequests asserts 429 Too Many Requests.
func (r *Response) StatusTooManyRequests() *Response {
	return r.Status(http.StatusTooManyRequests)
}

// StatusInternalServerError asserts 500 Internal Server Error.
func (r *Response) StatusInternalServerError() *Response {
	return r.Status(http.StatusInternalServerError)
}

// HeaderEquals asserts a header value.
func (r *Response) HeaderEquals(key, expected string) *Response {
	r.t.Helper()
	actual := r.Header.Get(key)
	if actual != expected {
		r.t.Errorf("expected header %s=%q, got %q", key, expected, actual)
	}
	return r
}

// HeaderContains asserts a header contains a substring.
func (r *Response) HeaderContains(key, substr string) *Response {
	r.t.Helper()
	actual := r.Header.Get(key)
	if !strings.Contains(actual, substr) {
		r.t.Errorf("expected header %s to contain %q, got %q", key, substr, actual)
	}
	return r
}

// HeaderExists asserts a header exists.
func (r *Response) HeaderExists(key string) *Response {
	r.t.Helper()
	if r.Header.Get(key) == "" {
		r.t.Errorf("expected header %s to exist", key)
	}
	return r
}

// ContentType asserts the Content-Type header.
func (r *Response) ContentType(expected string) *Response {
	return r.HeaderContains("Content-Type", expected)
}

// ContentTypeJSON asserts Content-Type is application/json.
func (r *Response) ContentTypeJSON() *Response {
	return r.ContentType("application/json")
}

// BodyEquals asserts the body equals the expected string.
func (r *Response) BodyEquals(expected string) *Response {
	r.t.Helper()
	actual := string(r.Body)
	if actual != expected {
		r.t.Errorf("expected body %q, got %q", expected, actual)
	}
	return r
}

// BodyContains asserts the body contains a substring.
func (r *Response) BodyContains(substr string) *Response {
	r.t.Helper()
	if !strings.Contains(string(r.Body), substr) {
		r.t.Errorf("expected body to contain %q, got %q", substr, string(r.Body))
	}
	return r
}

// BodyNotContains asserts the body does not contain a substring.
func (r *Response) BodyNotContains(substr string) *Response {
	r.t.Helper()
	if strings.Contains(string(r.Body), substr) {
		r.t.Errorf("expected body not to contain %q, got %q", substr, string(r.Body))
	}
	return r
}

// BodyEmpty asserts the body is empty.
func (r *Response) BodyEmpty() *Response {
	r.t.Helper()
	if len(r.Body) != 0 {
		r.t.Errorf("expected empty body, got %q", string(r.Body))
	}
	return r
}

// JSON unmarshals the body into v.
func (r *Response) JSON(v any) *Response {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		r.t.Fatalf("failed to unmarshal JSON: %v\nBody: %s", err, string(r.Body))
	}
	return r
}

// JSONPath extracts a value from the JSON body at the given path.
// Path is dot-separated (e.g., "user.name", "items.0.id").
func (r *Response) JSONPath(path string) any {
	r.t.Helper()

	var data any
	if err := json.Unmarshal(r.Body, &data); err != nil {
		r.t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				r.t.Fatalf("path %q not found at %q", path, part)
			}
		case []any:
			var idx int
			if err := json.Unmarshal([]byte(part), &idx); err != nil {
				r.t.Fatalf("invalid array index %q in path %q", part, path)
			}
			if idx < 0 || idx >= len(v) {
				r.t.Fatalf("array index %d out of bounds in path %q", idx, path)
			}
			current = v[idx]
		default:
			r.t.Fatalf("cannot navigate path %q at %q (type %T)", path, part, current)
		}
	}

	return current
}

// JSONPathEquals asserts a JSON path equals an expected value.
func (r *Response) JSONPathEquals(path string, expected any) *Response {
	r.t.Helper()
	actual := r.JSONPath(path)

	// Convert both to JSON for comparison
	actualJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)

	if string(actualJSON) != string(expectedJSON) {
		r.t.Errorf("expected %s=%v, got %v", path, expected, actual)
	}
	return r
}

// JSONPathContains asserts a JSON path contains a substring (for strings).
func (r *Response) JSONPathContains(path, substr string) *Response {
	r.t.Helper()
	actual := r.JSONPath(path)
	str, ok := actual.(string)
	if !ok {
		r.t.Fatalf("expected string at path %q, got %T", path, actual)
	}
	if !strings.Contains(str, substr) {
		r.t.Errorf("expected %s to contain %q, got %q", path, substr, str)
	}
	return r
}

// Print prints the response for debugging.
func (r *Response) Print() *Response {
	r.t.Logf("Status: %d\n", r.StatusCode)
	r.t.Logf("Headers: %v\n", r.Header)
	r.t.Logf("Body: %s\n", string(r.Body))
	return r
}

// String returns the response body as a string.
func (r *Response) String() string {
	return string(r.Body)
}

// Bytes returns the response body as bytes.
func (r *Response) Bytes() []byte {
	return r.Body
}
