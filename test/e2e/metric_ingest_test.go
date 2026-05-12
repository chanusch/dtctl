//go:build integration
// +build integration

package e2e

import (
	"strings"
	"testing"

	"github.com/dynatrace-oss/dtctl/pkg/resources/metric"
	"github.com/dynatrace-oss/dtctl/test/integration"
)

func TestMetricIngest_LineProtocol(t *testing.T) {
	env := integration.SetupIntegration(t)
	defer env.Cleanup.Cleanup(t)

	h := metric.NewHandler(env.Client)

	payload := strings.Join([]string{
		"dtctl.e2e.test.metric,host=e2e-test gauge,1.0",
		"dtctl.e2e.test.metric,host=e2e-test gauge,2.0",
	}, "\n") + "\n"

	result, err := h.IngestLine(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("IngestLine failed: %v", err)
	}
	if result.LinesOk != 2 {
		t.Errorf("expected linesOk=2, got %d", result.LinesOk)
	}
	if result.LinesInvalid != 0 {
		t.Errorf("expected linesInvalid=0, got %d", result.LinesInvalid)
	}
	if result.Chunks != 1 {
		t.Errorf("expected 1 chunk, got %d", result.Chunks)
	}
}

func TestMetricIngest_OTLPJson(t *testing.T) {
	env := integration.SetupIntegration(t)
	defer env.Cleanup.Cleanup(t)

	h := metric.NewHandler(env.Client)

	// Minimal valid OTLP/JSON ExportMetricsServiceRequest.
	payload := `{"resourceMetrics":[{"resource":{},"scopeMetrics":[{"scope":{},"metrics":[{"name":"dtctl.e2e.test.otlp","gauge":{"dataPoints":[{"asDouble":1.0,"timeUnixNano":"1700000000000000000"}]}}]}]}]}`

	result, err := h.IngestOTLP(strings.NewReader(payload), metric.OTLPJSON)
	if err != nil {
		t.Fatalf("IngestOTLP JSON failed: %v", err)
	}
	if result.BytesSent != len(payload) {
		t.Errorf("expected BytesSent=%d, got %d", len(payload), result.BytesSent)
	}
	if result.PartialRejected != 0 {
		t.Errorf("expected PartialRejected=0, got %d", result.PartialRejected)
	}
}
