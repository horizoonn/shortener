package response

import "net/http"

const statusCodeUninitialized = -1

type ResponseWriter struct {
	http.ResponseWriter
	statusCode     int
	headersWritten bool
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	if rw, ok := w.(*ResponseWriter); ok {
		return rw
	}

	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     statusCodeUninitialized,
	}
}

func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if !rw.headersWritten {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.ResponseWriter.Write(data)
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	if rw.headersWritten {
		return
	}

	rw.ResponseWriter.WriteHeader(statusCode)
	rw.statusCode = statusCode
	rw.headersWritten = true
}

func (rw *ResponseWriter) GetStatusCode() int {
	if rw.statusCode == statusCodeUninitialized {
		return http.StatusOK
	}

	return rw.statusCode
}

func (rw *ResponseWriter) HeadersWritten() bool {
	return rw.headersWritten
}
