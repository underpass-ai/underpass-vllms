package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type nonFlushingRecorder struct {
	header http.Header
	body   []byte
}

func (r *nonFlushingRecorder) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}
	return r.header
}

func (r *nonFlushingRecorder) Write(body []byte) (int, error) {
	r.body = append(r.body, body...)
	return len(body), nil
}

func (r *nonFlushingRecorder) WriteHeader(_ int) {}

type failingResponseWriter struct {
	*httptest.ResponseRecorder
	failWrite bool
}

func (w *failingResponseWriter) Write(body []byte) (int, error) {
	if w.failWrite {
		return 0, errors.New("write failed")
	}
	return w.ResponseRecorder.Write(body)
}

func TestSSEWriterStartAndDone(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := newSSEWriter(recorder)

	if err := writer.Data(map[string]string{"ok": "true"}); err != nil {
		t.Fatalf("unexpected data error: %v", err)
	}
	if err := writer.Done(); err != nil {
		t.Fatalf("unexpected done error: %v", err)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "data: [DONE]\n\n") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestSSEWriterRequiresFlusher(t *testing.T) {
	writer := newSSEWriter(&nonFlushingRecorder{})

	if err := writer.Start(); err == nil {
		t.Fatalf("expected flusher error")
	}
}

func TestSSEWriterPropagatesWriteErrors(t *testing.T) {
	recorder := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder(), failWrite: true}
	writer := newSSEWriter(recorder)

	if err := writer.Data(map[string]string{"ok": "true"}); err == nil {
		t.Fatalf("expected write error")
	}
	if err := writer.Done(); err == nil {
		t.Fatalf("expected write error on done")
	}
}
