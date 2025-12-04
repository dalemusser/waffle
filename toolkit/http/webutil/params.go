// toolkit/http/webutil/params.go
package webutil

import "net/url"

// AddOrSetQueryParams appends or overwrites query params on a base URL string.
// On parse failure it returns the original string unchanged.
func AddOrSetQueryParams(base string, kv map[string]string) string {
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, v := range kv {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
