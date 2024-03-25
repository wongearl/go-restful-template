package net

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultValue(t *testing.T) {
	httpTest := HTTPTest{}
	httpTest = httpTest.SetDefaultValue()
	assert.Equal(t, http.MethodGet, httpTest.Method)
	assert.Equal(t, http.StatusOK, httpTest.ExpectCode)
	assert.Equal(t, ContentTypeJSON, httpTest.ContentType)

	httpTestHasVal := HTTPTest{
		Method:      http.MethodPost,
		ExpectCode:  http.StatusNotFound,
		ContentType: "fake",
	}
	httpTestHasValNew := httpTestHasVal.SetDefaultValue()
	assert.Equal(t, httpTestHasVal, httpTestHasValNew)

	// test method GetBody
	forBodyTest := HTTPTest{
		Body: "fake",
	}
	body := forBodyTest.GetBody()
	assert.NotNil(t, body)
	data, err := io.ReadAll(body)
	assert.Nil(t, err)
	assert.Equal(t, "fake", string(data))
}

func TestGetSampleFileForUpload(t *testing.T) {
	data := GetSampleFileForUpload("", map[string]string{
		"name": "target",
		"os":   "ubuntu",
	})
	assert.Contains(t, string(data), "Content-Disposition: form-data")
	assert.Contains(t, string(data), "Content-Type: application/octet-stream")
	assert.Contains(t, string(data), `form-data; name="name"`)
	assert.True(t, strings.HasPrefix(string(data), "--"))
}

func TestAddHeaders(t *testing.T) {
	httpTest := HTTPTest{
		Headers: map[string]string{
			"key": "value",
		},
		ContentType: "application/json",
	}
	headers := http.Header{}
	httpTest.AddHeaders(headers)
	assert.Equal(t, "value", headers.Get("key"))
	assert.Equal(t, "application/json", headers.Get("Content-type"))
}
