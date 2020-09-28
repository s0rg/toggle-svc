package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
)

type handler func(ctx context.Context, w io.Writer, r *http.Request) error

func wrapAPI(name string, h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer

		code := http.StatusMethodNotAllowed

		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(code), code)

			return
		}

		code = http.StatusBadRequest

		if err := h(r.Context(), &buf, r); err != nil {
			if !errors.Is(err, errBadRequest) {
				code = http.StatusInternalServerError

				log.Println("api:", name, "error:", err)
			}

			http.Error(w, http.StatusText(code), code)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if _, err := buf.WriteTo(w); err != nil {
			log.Println("api:", name, "response error:", err)
		}
	}
}
