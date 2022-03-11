package _http

import "net/http"

type (
	HTTPError struct {
		error
		StatusCode int
	}
	WithHeader interface {
		Header() http.Header
	}
)

func NewErr(code int, err error) HTTPError {
	return HTTPError{
		error:      err,
		StatusCode: code,
	}
}
