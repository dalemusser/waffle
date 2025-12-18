// pantry/httpnav/back.go
package httpnav

import (
	"net/http"
	"strings"
)

func HasExplicitReturn(r *http.Request) bool {
	ret := strings.TrimSpace(r.URL.Query().Get("return"))
	return ret != "" && strings.HasPrefix(ret, "/")
}
