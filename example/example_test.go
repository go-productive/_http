package example

import (
	"github.com/go-productive/_http"
	"testing"
)

func TestExample(t *testing.T) {
	server := _http.New()
	e := new(Example)
	e.RegisterRoute(server)
	panic(server.Engine.Run(":65000"))
}
