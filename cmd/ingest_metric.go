package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dynatrace-oss/dtctl/pkg/output"
	"github.com/dynatrace-oss/dtctl/pkg/resources/metric"
	"github.com/dynatrace-oss/dtctl/pkg/safety"
)

var ingestMetricCmd = &cobra.Command{
	Use:   "metric [-f FILE]",
	Short: "Ingest metric data points into Dynatrace",
	Long: `Ingest metric data points into Dynatrace.

Line protocol (default):
  One metric per line: <key>[,dim=val,...] [<type>,]<value> [<timestamp_ms>]
  Limits: 1 000 data lines and 1 MB per POST; larger inputs are auto-chunked.

OTLP passthrough (--format otlp-json or otlp-proto):
  The payload is forwarded verbatim to the OTLP metrics endpoint.
  No chunking is performed; payload size limits are enforced server-side.

Required scope: metrics.ingest (classic token) or storage:metrics:write (platform token).`,
	Example: `  # Push from a line-protocol file
  dtctl ingest metric -f metrics.lp

  # Push from stdin (pipe)
  echo 'my.metric,host=demo gauge,42' | dtctl ingest metric

  # Push an OTLP/JSON payload
  dtctl ingest metric -f payload.json --format otlp-json

  # Push an OTLP/protobuf payload
  dtctl ingest metric -f payload.pb --format otlp-proto

  # Dry run: show what would be sent without POSTing
  dtctl ingest metric -f metrics.lp --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filename, _ := cmd.Flags().GetString("filename")
		formatStr, _ := cmd.Flags().GetString("format")

		ingestFormat, err := parseIngestFormat(formatStr)
		if err != nil {
			return err
		}

		// Resolve the input reader.
		var reader io.Reader
		var sourceName string
		if filename == "" || filename == "-" {
			if isTerminal(os.Stdin) {
				return fmt.Errorf("no input: use -f <file> or pipe data via stdin")
			}
			reader = os.Stdin
			sourceName = "stdin"
		} else {
			f, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", filename, err)
			}
			defer f.Close()
			reader = f
			sourceName = filename
		}

		// Dry-run: report intent without posting.
		if dryRun {
			return runDryRun(reader, sourceName, ingestFormat)
		}

		_, c, err := SetupWithSafety(safety.OperationCreate)
		if err != nil {
			return err
		}

		printer := NewPrinter()
		enrichAgent(printer, "ingest", "metric")

		h := metric.NewHandler(c)

		if ingestFormat != metric.OTLPLine {
			result, err := h.IngestOTLP(reader, ingestFormat)
			if err != nil {
				// Surface partial message but keep the result visible.
				output.PrintWarning("%s", err)
				if result != nil {
					return printer.Print(result)
				}
				return err
			}
			return printer.Print(result)
		}

		// Line protocol
		result, err := h.IngestLine(reader)
		if errors.Is(err, metric.ErrPartialIngest) {
			output.PrintWarning("some lines were rejected (%d invalid)", result.LinesInvalid)
			if err2 := printer.Print(result); err2 != nil {
				return err2
			}
			return err // exit non-zero
		}
		if err != nil {
			return err
		}
		if result.Chunks == 0 {
			output.PrintInfo("No data lines found in input")
			return nil
		}
		return printer.Print(result)
	},
}

func init() {
	ingestMetricCmd.Flags().StringP("filename", "f", "", "file containing metric data ('-' for stdin)")
	ingestMetricCmd.Flags().String("format", "line", "payload format: line, otlp-json, or otlp-proto")
}

// parseIngestFormat converts the --format flag value to an OTLPFormat or
// sentinel indicating line protocol.
func parseIngestFormat(s string) (metric.OTLPFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "line", "":
		return metric.OTLPLine, nil
	case "otlp-json":
		return metric.OTLPJSON, nil
	case "otlp-proto":
		return metric.OTLPProto, nil
	default:
		return 0, fmt.Errorf("unknown format %q: must be one of line, otlp-json, otlp-proto", s)
	}
}

// runDryRun prints what would be sent without making any HTTP calls.
func runDryRun(r io.Reader, sourceName string, format metric.OTLPFormat) error {
	payload, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if format != metric.OTLPLine {
		fmt.Printf("Dry run: would POST %d bytes to OTLP metrics endpoint\n", len(payload))
		fmt.Printf("Source:       %s\n", sourceName)
		fmt.Printf("Content-Type: %s\n", format.ContentType())
		fmt.Printf("Endpoint:     /platform/classic/environment-api/v2/otlp/v1/metrics\n")
		return nil
	}

	// Count data lines and estimate chunks for line protocol.
	lines := strings.Split(string(payload), "\n")
	dataLines, totalBytes := 0, len(payload)
	var firstDataLine string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			dataLines++
			if firstDataLine == "" {
				firstDataLine = l
			}
		}
	}
	chunks := dataLines / 1000
	if dataLines%1000 > 0 || dataLines == 0 {
		chunks++
	}
	// Account for byte-limit splits (approximate).
	if totalBytes > 1_000_000 {
		byteChunks := totalBytes / 1_000_000
		if totalBytes%1_000_000 > 0 {
			byteChunks++
		}
		if byteChunks > chunks {
			chunks = byteChunks
		}
	}

	fmt.Printf("Dry run: would POST %d data lines in %d chunk(s) to metrics ingest endpoint\n", dataLines, chunks)
	fmt.Printf("Source:    %s\n", sourceName)
	fmt.Printf("Total:     %d bytes\n", totalBytes)
	if firstDataLine != "" {
		fmt.Printf("First line: %s\n", firstDataLine)
	}
	return nil
}
