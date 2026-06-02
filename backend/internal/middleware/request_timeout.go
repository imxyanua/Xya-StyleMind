package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		baseWriter := c.Writer
		timeoutWriter := newTimeoutResponseWriter(baseWriter)
		c.Writer = timeoutWriter

		done := make(chan struct{})
		panicChan := make(chan any, 1)
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panicChan <- recovered
				}
				close(done)
			}()
			c.Next()
		}()

		select {
		case recovered := <-panicChan:
			c.Writer = baseWriter
			panic(recovered)
		case <-done:
			timeoutWriter.flush()
			c.Writer = baseWriter
		case <-ctx.Done():
			timeoutWriter.writeTimeout()
			c.Abort()
		}
	}
}

type timeoutResponseWriter struct {
	gin.ResponseWriter

	mu       sync.Mutex
	header   http.Header
	body     bytes.Buffer
	status   int
	size     int
	written  bool
	timedOut bool
}

func newTimeoutResponseWriter(base gin.ResponseWriter) *timeoutResponseWriter {
	header := make(http.Header, len(base.Header()))
	for key, values := range base.Header() {
		header[key] = append([]string(nil), values...)
	}
	return &timeoutResponseWriter{
		ResponseWriter: base,
		header:         header,
		status:         http.StatusOK,
	}
}

func (w *timeoutResponseWriter) Header() http.Header {
	return w.header
}

func (w *timeoutResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.written || w.timedOut {
		return
	}
	w.status = code
	w.written = true
}

func (w *timeoutResponseWriter) WriteHeaderNow() {
	w.WriteHeader(w.status)
}

func (w *timeoutResponseWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return 0, context.DeadlineExceeded
	}
	if !w.written {
		w.status = http.StatusOK
		w.written = true
	}
	n, err := w.body.Write(data)
	w.size += n
	return n, err
}

func (w *timeoutResponseWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

func (w *timeoutResponseWriter) Status() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

func (w *timeoutResponseWriter) Size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}

func (w *timeoutResponseWriter) Written() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.written || w.timedOut
}

func (w *timeoutResponseWriter) Flush() {}

func (w *timeoutResponseWriter) flush() {
	w.mu.Lock()
	if w.timedOut {
		w.mu.Unlock()
		return
	}
	header := cloneHeader(w.header)
	status := w.status
	body := append([]byte(nil), w.body.Bytes()...)
	written := w.written
	w.mu.Unlock()

	copyHeader(w.ResponseWriter.Header(), header)
	if written {
		w.ResponseWriter.WriteHeader(status)
		if len(body) > 0 {
			_, _ = w.ResponseWriter.Write(body)
		}
	}
}

func (w *timeoutResponseWriter) writeTimeout() {
	payload, _ := json.Marshal(response.APIResponse{
		Success: false,
		Message: "request timeout",
	})

	w.mu.Lock()
	w.timedOut = true
	w.status = http.StatusGatewayTimeout
	w.size = len(payload)
	w.mu.Unlock()

	w.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
	_, _ = w.ResponseWriter.Write(payload)
}

func cloneHeader(source http.Header) http.Header {
	header := make(http.Header, len(source))
	for key, values := range source {
		header[key] = append([]string(nil), values...)
	}
	return header
}

func copyHeader(destination, source http.Header) {
	for key, values := range source {
		destination.Del(key)
		for _, value := range values {
			destination.Add(key, value)
		}
	}
}
