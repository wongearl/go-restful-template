package net

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// HTTPTest represents a common HTTP testing item
type HTTPTest struct {
	Method      string
	API         string
	Body        string
	Headers     map[string]string
	ContentType string
	ExpectCode  int
}

// SetDefaultValue
func (t HTTPTest) SetDefaultValue() HTTPTest {
	if t.Method == "" {
		t.Method = http.MethodGet
	}
	if t.ExpectCode == 0 {
		t.ExpectCode = http.StatusOK
	}
	if t.ContentType == "" {
		t.ContentType = ContentTypeJSON
	}
	return t
}

// GetBody converts body from string to io.Reader
func (t HTTPTest) GetBody() io.Reader {
	body := &bytes.Buffer{}
	_, _ = body.WriteString(t.Body)
	return body
}

// AddHeaders adds headers
func (t HTTPTest) AddHeaders(headers http.Header) {
	for key, val := range t.Headers {
		headers.Set(key, val)
	}
	if t.ContentType != "" {
		headers.Set("Content-type", t.ContentType)
	}
}

// Assert assert the response body
func (t HTTPTest) Assert(tt *testing.T, resp *httptest.ResponseRecorder) (data []byte) {
	data = resp.Body.Bytes()
	assert.Equal(tt, t.ExpectCode, resp.Code, string(data))
	return
}

// GetSampleFileForUpload returns a sample data for file upload
func GetSampleFileForUpload(data string, form map[string]string) (body []byte) {
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	_ = writer.SetBoundary("abcd")

	for key, val := range form {
		_ = writer.WriteField(key, val)
	}

	fileWriter, _ := writer.CreateFormFile("filename", "filename")
	_, _ = io.Copy(fileWriter, bytes.NewBufferString(data))
	_ = writer.Close()
	body = buf.Bytes()
	return
}

const (
	// TestDefaultNamespace is the default namespace name for unit test
	TestDefaultNamespace = "default"
)
