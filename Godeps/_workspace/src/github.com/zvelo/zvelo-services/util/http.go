package util

import (
	"net/http"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/unrolled/render"
)

var (
	r = render.New()
)

// RenderPB renders a proto.Message properly to an http.ResponseWriter
func RenderPB(w http.ResponseWriter, status int, msg proto.Message) {
	pb, err := proto.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/x-protobuf")
	w.WriteHeader(status)
	w.Write(pb)
}

// Render looks for a ".pb" extension and renders either the protobuf or JSON
// message properly to an http.ResponseWriter
func Render(w http.ResponseWriter, req *http.Request, status int, data interface{}) {
	if msg, ok := data.(proto.Message); ok && filepath.Ext(req.URL.Path) == ".pb" {
		RenderPB(w, status, msg)
		return
	}

	if callback := req.FormValue("callback"); len(callback) > 0 {
		r.JSONP(w, status, callback, data)
		return
	}

	r.JSON(w, status, data)
}
