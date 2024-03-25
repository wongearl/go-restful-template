package filters

import (
	"fmt"
	"net/http"

	"github.com/wongearl/go-restful-template/pkg/aiserver/request"
	"github.com/wongearl/go-restful-template/pkg/utils/stringutils"

	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
)

func WithRequestInfo(handler http.Handler, resolver request.RequestInfoResolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ws := req.Header.Get("Upgrade")
		if ws == "websocket" {
			authorization := req.Header.Get("Authorization")
			if len(authorization) == 0 {
				xAuthorization := stringutils.GetAuthorizationFromCookie(req.Header.Get("Cookie"))
				if len(xAuthorization) != 0 {
					req.Header.Set("Authorization", xAuthorization)
				}
			}
		}

		ctx := req.Context()
		info, err := resolver.NewRequestInfo(req)
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("failed to create RequestInfo: %v", err))
			return
		}

		req = req.WithContext(request.WithRequestInfo(ctx, info))
		handler.ServeHTTP(w, req)
	})
}
