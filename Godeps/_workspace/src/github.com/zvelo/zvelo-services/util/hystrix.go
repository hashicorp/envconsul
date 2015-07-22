package util

import (
	"net/http"

	"github.com/afex/hystrix-go/hystrix"
)

// StartHystrixStreamHandler starts a hystrix stream server on addr. It is
// usually run in a separate goroutine.
func StartHystrixStreamHandler(addr string) {
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	http.ListenAndServe(addr, hystrixStreamHandler)
}
