package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin/render"
)

type NoEscapeJSON struct {
	Data any
}

func (r NoEscapeJSON) Render(w http.ResponseWriter) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r.Data)
}

func (r NoEscapeJSON) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

var _ render.Render = NoEscapeJSON{} // interface check
