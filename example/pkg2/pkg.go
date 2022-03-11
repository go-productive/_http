package pkg2

import (
	"net/http"
)

type (
	HelloResponse struct {
		Msg string `json:"msg"`
	}
)

func (h *HelloResponse) Header() http.Header {
	return http.Header{
		"AAA": []string{"BBB"},
	}
}
