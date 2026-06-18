package response

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

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

func (rw *ResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (rw *ResponseWriter) Flush() {
	if !rw.headersWritten {
		rw.WriteHeader(http.StatusOK)
	}

	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking: %w", http.ErrNotSupported)
	}

	return hijacker.Hijack()
}

func (rw *ResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	if !rw.headersWritten {
		rw.WriteHeader(http.StatusOK)
	}

	if readerFrom, ok := rw.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(reader)
	}

	return io.Copy(rw.ResponseWriter, reader)
}

func (rw *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := rw.ResponseWriter.(http.Pusher)
	if !ok {
		return fmt.Errorf("response writer does not support server push: %w", http.ErrNotSupported)
	}

	return pusher.Push(target, opts)
}
