package handlers

import "net/http"

// NotFoundPage serves the page shell with a 404 status, preserving render errors.
func NotFoundPage(shell http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &notFoundPageWriter{ResponseWriter: w}
		shell.ServeHTTP(response, r)
		response.WriteHeader(http.StatusNotFound)
	})
}

type notFoundPageWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *notFoundPageWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	if status < http.StatusBadRequest {
		status = http.StatusNotFound
	}
	w.Header().Set("Cache-Control", "no-store")
	w.ResponseWriter.WriteHeader(status)
}

func (w *notFoundPageWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusNotFound)
	}
	return w.ResponseWriter.Write(body)
}
