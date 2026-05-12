package cmd

import (
	"github.com/spf13/cobra"
)

// ingestCmd is the top-level verb for sending telemetry data to Dynatrace.
var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest telemetry data into Dynatrace",
	Long: `Ingest telemetry data into Dynatrace.

Supported resources:
  metric    Ingest metric data points (line protocol or OTLP passthrough)`,
	Example: `  # Push metric data points from a file
  dtctl ingest metric -f metrics.lp

  # Push from stdin
  echo 'my.metric,host=demo gauge,42' | dtctl ingest metric

  # Push an OTLP/JSON payload
  dtctl ingest metric -f payload.json --format otlp-json`,
	RunE: requireSubcommand,
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.AddCommand(ingestMetricCmd)
}
