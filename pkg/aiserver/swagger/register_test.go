package swagger

import (
	"fmt"
	"github.com/wongearl/go-restful-template/pkg/utils/net"
	"net/http"
	"net/http/httptest"
	"testing"

	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
)

func TestAddToContainer(t *testing.T) {
	tests := []struct {
		name     string
		rootDir  string
		httpTest net.HTTPTest
	}{{
		name: "root",
		httpTest: net.HTTPTest{
			API:        "/",
			ExpectCode: http.StatusPermanentRedirect,
		},
	}, {
		name: "root with url query",
		httpTest: net.HTTPTest{
			API: "/?url=http://localhost/apidocs.json",
		},
	}, {
		name: "request swagger-ui.js",
		httpTest: net.HTTPTest{
			API: "/swagger-ui.js",
		},
	}, {
		name:    "wrong root dir",
		rootDir: ".",
		httpTest: net.HTTPTest{
			API:        "/swagger-ui.js",
			ExpectCode: http.StatusNotFound,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rootDir == "" {
				tt.rootDir = "../../../docs/swagger-ui"
			}
			container := restful.NewContainer()
			AddToContainer(tt.rootDir, container)

			writer := httptest.NewRecorder()
			tt.httpTest = tt.httpTest.SetDefaultValue()
			req := httptest.NewRequest(tt.httpTest.Method,
				fmt.Sprintf("http://localhost/apidocs%s", tt.httpTest.API), nil)

			container.Dispatch(writer, req)
			assert.Equal(t, tt.httpTest.ExpectCode, writer.Code)
		})
	}
}
