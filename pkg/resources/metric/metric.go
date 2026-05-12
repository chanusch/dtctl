package metric

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dynatrace-oss/dtctl/pkg/client"
)

const (
	lineProtocolEndpoint = "/platform/classic/environment-api/v2/metrics/ingest"
	otlpEndpoint         = "/platform/classic/environment-api/v2/otlp/v1/metrics"

	maxChunkLines = 1000
	maxChunkBytes = 1_000_000 // 1 MB
)

// OTLPFormat selects the wire format for OTLP passthrough.
type OTLPFormat int

const (
	// OTLPLine is a sentinel used by the cmd layer to indicate line-protocol
	// input (not an OTLP format; the handler ignores this value).
	OTLPLine  OTLPFormat = -1
	OTLPJSON  OTLPFormat = iota
	OTLPProto OTLPFormat = iota
)

func (f OTLPFormat) ContentType() string {
	if f == OTLPProto {
		return "application/x-protobuf"
	}
	return "application/json"
}

// ErrPartialIngest is returned when the API accepted the request but some
// lines were invalid. IngestResult still holds the full summary.
var ErrPartialIngest = errors.New("partial ingest: some lines were rejected")

// LineError describes a single rejected line from the API.
type LineError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// LineResult summarises the outcome of a line-protocol ingest.
type LineResult struct {
	LinesOk      int           `json:"linesOk"      table:"LINES_OK"`
	LinesInvalid int           `json:"linesInvalid" table:"LINES_INVALID"`
	Chunks       int           `json:"chunks"       table:"CHUNKS"`
	Duration     time.Duration `json:"duration"     table:"DURATION"`
	Errors       []LineError   `json:"errors,omitempty" table:"-"`
}

// OTLPResult summarises the outcome of an OTLP passthrough ingest.
type OTLPResult struct {
	BytesSent       int           `json:"bytesSent"       table:"BYTES_SENT"`
	PartialRejected int64         `json:"partialRejected" table:"PARTIAL_REJECTED"`
	Duration        time.Duration `json:"duration"        table:"DURATION"`
	PartialMessage  string        `json:"partialMessage,omitempty" table:"-"`
}

// Handler handles metric ingestion.
type Handler struct {
	client *client.Client
}

// NewHandler creates a new metric handler.
func NewHandler(c *client.Client) *Handler {
	return &Handler{client: c}
}

// lineIngestResponse matches the 202 body returned by /metrics/ingest.
type lineIngestResponse struct {
	LinesOk      int         `json:"linesOk"`
	LinesInvalid int         `json:"linesInvalid"`
	Error        interface{} `json:"error"`
}

// IngestLine reads line-protocol metric data from r, chunks it (≤1000 data
// lines and ≤1 MB per POST), and sends each chunk to the Metrics API v2.
// Blank lines and comment lines (starting with #) are counted for byte budget
// but not toward the 1000-line limit, and are forwarded verbatim so that
// server-side error line numbers remain aligned with the source.
func (h *Handler) IngestLine(r io.Reader) (*LineResult, error) {
	start := time.Now()
	result := &LineResult{}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxChunkBytes+65536), maxChunkBytes+65536)

	var (
		chunk     bytes.Buffer
		chunkData int // data lines in current chunk (excludes blanks/comments)
	)

	flush := func() error {
		if chunk.Len() == 0 {
			return nil
		}
		resp, err := h.client.HTTP().R().
			SetHeader("Content-Type", "text/plain; charset=utf-8").
			SetBody(chunk.String()).
			Post(lineProtocolEndpoint)

		chunk.Reset()
		chunkData = 0
		result.Chunks++

		if err != nil {
			return fmt.Errorf("failed to POST metrics chunk %d: %w", result.Chunks, err)
		}
		if resp.IsError() {
			return fmt.Errorf("failed to ingest metrics chunk %d: status %d: %s", result.Chunks, resp.StatusCode(), resp.String())
		}

		var chunkResp lineIngestResponse
		if err := json.Unmarshal(resp.Body(), &chunkResp); err != nil {
			return fmt.Errorf("failed to parse ingest response for chunk %d: %w", result.Chunks, err)
		}
		result.LinesOk += chunkResp.LinesOk
		result.LinesInvalid += chunkResp.LinesInvalid
		if chunkResp.Error != nil {
			result.Errors = append(result.Errors, LineError{
				Line:    result.LinesOk + result.LinesInvalid,
				Message: fmt.Sprintf("%v", chunkResp.Error),
			})
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		isData := !isBlankOrComment(line)
		lineBytes := len(line) + 1 // +1 for newline

		// Flush if adding this line would exceed either limit.
		if chunk.Len() > 0 && (chunk.Len()+lineBytes > maxChunkBytes || (isData && chunkData >= maxChunkLines)) {
			if err := flush(); err != nil {
				return result, err
			}
		}

		chunk.WriteString(line)
		chunk.WriteByte('\n')
		if isData {
			chunkData++
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("error reading input: %w", err)
	}

	if err := flush(); err != nil {
		return result, err
	}

	result.Duration = time.Since(start)

	if result.LinesInvalid > 0 {
		return result, ErrPartialIngest
	}
	return result, nil
}

// IngestOTLP reads a pre-built OTLP/HTTP payload from r and forwards it
// verbatim to the OTLP metrics endpoint. No chunking is performed.
func (h *Handler) IngestOTLP(r io.Reader, format OTLPFormat) (*OTLPResult, error) {
	start := time.Now()

	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read OTLP payload: %w", err)
	}

	result := &OTLPResult{BytesSent: len(payload)}

	resp, err := h.client.HTTP().R().
		SetHeader("Content-Type", format.ContentType()).
		SetBody(payload).
		Post(otlpEndpoint)

	result.Duration = time.Since(start)

	if err != nil {
		return result, fmt.Errorf("failed to POST OTLP metrics: %w", err)
	}
	if resp.IsError() {
		return result, fmt.Errorf("failed to ingest OTLP metrics: status %d: %s", resp.StatusCode(), resp.String())
	}

	// Decode partial_success from the response (best-effort; never fail on parse errors).
	switch format {
	case OTLPJSON:
		result.PartialRejected, result.PartialMessage = decodeOTLPJSONResponse(resp.Body())
	case OTLPProto:
		result.PartialRejected, result.PartialMessage = decodeOTLPProtoResponse(resp.Body())
	}

	if result.PartialRejected > 0 {
		return result, fmt.Errorf("partial ingest: %d datapoints rejected: %s", result.PartialRejected, result.PartialMessage)
	}
	return result, nil
}

// isBlankOrComment returns true for empty lines and lines starting with '#'.
func isBlankOrComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}
