package logrusmiddleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	Middleware struct {
		Logger logrus.FieldLogger
		Name string
	}

	Handler struct {
		rw        http.ResponseWriter
		status    int
		size      int
		m         *Middleware
		handler   http.Handler
		component string
	}
)

// Create a new handler.
func (m *Middleware) Handler(h http.Handler, component string) *Handler {
	return &Handler{
		m:         m,
		handler:   h,
		component: component,
	}
}

// Wrapper for the "real" ResponseWriter.Write
func (h *Handler) Write(b []byte) (int, error) {
	if h.status == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		h.status = http.StatusOK
	}
	size, err := h.rw.Write(b)
	h.size += size
	return size, err
}

// Wrapper around ResponseWriter.WriteHeader
func (h *Handler) WriteHeader(s int) {
	h.rw.WriteHeader(s)
	h.status = s
}

// Wrapper around ResponseWriter.Header
func (h *Handler) Header() http.Header {
	return h.rw.Header()
}

// ServeHTTP calls the "real" handler and logs using the logger
func (h *Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	start := time.Now()

	h.rw = rw
	h.handler.ServeHTTP(h, r)

	latency := time.Since(start)

	fields := logrus.Fields{
		"status":     h.status,
		"method":     r.Method,
		"request":    r.RequestURI,
		"remote":     r.RemoteAddr,
		"duration":   float64(latency.Nanoseconds()) / float64(1000),
		"size":       h.size,
		"referer":    r.Referer(),
		"user-agent": r.UserAgent(),
	}

	if h.m.Name != "" {
		fields["name"] = h.m.Name
	}

	if h.component != "" {
		fields["component"] = h.component
	}

	if l := h.m.Logger; l != nil {
		l.WithFields(fields).Info("completed handling request")
	} else {
		logrus.WithFields(fields).Info("completed handling request")
	}
}
