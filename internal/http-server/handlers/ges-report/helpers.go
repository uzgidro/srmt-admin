package gesreport

import (
	"fmt"
	"net/http"
	"strconv"
)

func parseIntParam(r *http.Request, name string) (int64, error) {
	s := r.URL.Query().Get(name)
	if s == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	return strconv.ParseInt(s, 10, 64)
}
