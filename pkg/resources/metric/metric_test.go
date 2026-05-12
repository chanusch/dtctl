package metric

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dynatrace-oss/dtctl/pkg/client"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Handler) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := client.NewForTesting(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return srv, NewHandler(c)
}

func lineResp(ok, invalid int, errMsg string) []byte {
	type errObj struct {
		Message string `json:"message"`
	}
	type resp struct {
		LinesOk      int     `json:"linesOk"`
		LinesInvalid int     `json:"linesInvalid"`
		Error        *errObj `json:"error"`
	}
	r := resp{LinesOk: ok, LinesInvalid: invalid}
	if errMsg != "" {
		r.Error = &errObj{Message: errMsg}
	}
	b, _ := json.Marshal(r)
	return b
}

// TestIngestLine_SmallPayload verifies a single-chunk successful ingest.
func TestIngestLine_SmallPayload(t *testing.T) {
	calls := 0
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != lineProtocolEndpoint {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write(lineResp(3, 0, ""))
	})

	input := "my.metric,host=h1 gauge,1\nmy.metric,host=h2 gauge,2\nmy.metric,host=h3 gauge,3\n"
	result, err := h.IngestLine(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 POST, got %d", calls)
	}
	if result.Chunks != 1 {
		t.Errorf("expected 1 chunk, got %d", result.Chunks)
	}
	if result.LinesOk != 3 {
		t.Errorf("expected 3 linesOk, got %d", result.LinesOk)
	}
	if result.LinesInvalid != 0 {
		t.Errorf("expected 0 linesInvalid, got %d", result.LinesInvalid)
	}
}

// TestIngestLine_EmptyInput verifies no HTTP call is made for empty input.
func TestIngestLine_EmptyInput(t *testing.T) {
	calls := 0
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
	})

	result, err := h.IngestLine(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Errorf("expected 0 POSTs for empty input, got %d", calls)
	}
	if result.Chunks != 0 {
		t.Errorf("expected 0 chunks, got %d", result.Chunks)
	}
}

// TestIngestLine_CommentsAndBlankLines verifies blank/comment lines are skipped
// in the data-line count but preserved in the payload.
func TestIngestLine_CommentsAndBlankLines(t *testing.T) {
	var body string
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(http.StatusAccepted)
		w.Write(lineResp(1, 0, ""))
	})

	input := "# a comment\n\nmy.metric gauge,1\n"
	_, err := h.IngestLine(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "# a comment") {
		t.Error("expected comment line to be forwarded verbatim")
	}
}

// TestIngestLine_ChunksOnLineLimit verifies that payloads exceeding 1000 data
// lines are split into multiple POST requests.
func TestIngestLine_ChunksOnLineLimit(t *testing.T) {
	calls := 0
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusAccepted)
		w.Write(lineResp(maxChunkLines, 0, ""))
	})

	var sb strings.Builder
	for i := 0; i < maxChunkLines+1; i++ {
		fmt.Fprintf(&sb, "my.metric,i=%d gauge,%d\n", i, i)
	}

	result, err := h.IngestLine(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 POSTs for %d lines, got %d", maxChunkLines+1, calls)
	}
	if result.Chunks != 2 {
		t.Errorf("expected 2 chunks, got %d", result.Chunks)
	}
}

// TestIngestLine_PartialIngest verifies that linesInvalid > 0 returns ErrPartialIngest.
func TestIngestLine_PartialIngest(t *testing.T) {
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write(lineResp(2, 1, "invalid metric key"))
	})

	input := "my.metric gauge,1\nmy.metric gauge,2\nbad line\n"
	result, err := h.IngestLine(strings.NewReader(input))
	if !errors.Is(err, ErrPartialIngest) {
		t.Fatalf("expected ErrPartialIngest, got %v", err)
	}
	if result.LinesOk != 2 {
		t.Errorf("expected 2 linesOk, got %d", result.LinesOk)
	}
	if result.LinesInvalid != 1 {
		t.Errorf("expected 1 linesInvalid, got %d", result.LinesInvalid)
	}
}

// TestIngestLine_ServerError verifies that a non-2xx response returns an error.
func TestIngestLine_ServerError(t *testing.T) {
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	})

	_, err := h.IngestLine(strings.NewReader("my.metric gauge,1\n"))
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Errorf("expected 'status 401' in error, got: %v", err)
	}
}

// TestIngestOTLP_JSON verifies OTLP/JSON passthrough sends the right path and content-type.
func TestIngestOTLP_JSON(t *testing.T) {
	var gotPath, gotCT string
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	payload := `{"resourceMetrics":[]}`
	result, err := h.IngestOTLP(strings.NewReader(payload), OTLPJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != otlpEndpoint {
		t.Errorf("expected path %q, got %q", otlpEndpoint, gotPath)
	}
	if gotCT != "application/json" {
		t.Errorf("expected content-type 'application/json', got %q", gotCT)
	}
	if result.BytesSent != len(payload) {
		t.Errorf("expected BytesSent=%d, got %d", len(payload), result.BytesSent)
	}
}

// TestIngestOTLP_Proto verifies OTLP/proto passthrough uses the protobuf content-type.
func TestIngestOTLP_Proto(t *testing.T) {
	var gotCT string
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{}) // empty proto response (no partial_success)
	})

	payload := []byte{0x0a, 0x00} // minimal proto bytes
	result, err := h.IngestOTLP(strings.NewReader(string(payload)), OTLPProto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCT != "application/x-protobuf" {
		t.Errorf("expected 'application/x-protobuf', got %q", gotCT)
	}
	if result.PartialRejected != 0 {
		t.Errorf("expected 0 rejected, got %d", result.PartialRejected)
	}
}

// TestIngestOTLP_PartialSuccess verifies that a JSON response with partial_success
// populates PartialRejected and returns an error.
func TestIngestOTLP_PartialSuccess(t *testing.T) {
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"partialSuccess":{"rejectedDataPoints":3,"errorMessage":"bad type"}}`))
	})

	_, err := h.IngestOTLP(strings.NewReader(`{}`), OTLPJSON)
	if err == nil {
		t.Fatal("expected error for partial rejection, got nil")
	}
	if !strings.Contains(err.Error(), "3 datapoints rejected") {
		t.Errorf("expected rejection count in error, got: %v", err)
	}
}

// TestIngestOTLP_ServerError verifies that a non-2xx response returns an error.
func TestIngestOTLP_ServerError(t *testing.T) {
	_, h := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	})

	_, err := h.IngestOTLP(strings.NewReader(`{}`), OTLPJSON)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Errorf("expected 'status 403' in error, got: %v", err)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
