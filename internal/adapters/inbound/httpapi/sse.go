package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	started bool
}

func newSSEWriter(w http.ResponseWriter) *sseWriter {
	flusher, _ := w.(http.Flusher)
	return &sseWriter{w: w, flusher: flusher}
}

func (s *sseWriter) Start() error {
	if s.flusher == nil {
		return fmt.Errorf("streaming is not supported by the response writer")
	}
	if s.started {
		return nil
	}
	headers := s.w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")
	s.w.WriteHeader(http.StatusOK)
	s.flusher.Flush()
	s.started = true
	return nil
}

func (s *sseWriter) Data(payload any) error {
	if err := s.Start(); err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", body); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *sseWriter) Done() error {
	if err := s.Start(); err != nil {
		return err
	}
	if _, err := fmt.Fprint(s.w, "data: [DONE]\n\n"); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func newStreamingRequestID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("stream-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}
