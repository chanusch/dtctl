package metric

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protowire"
)

// decodeOTLPJSONResponse extracts partial_success fields from an
// ExportMetricsServiceResponse JSON body. Returns zeros on any parse error.
func decodeOTLPJSONResponse(body []byte) (rejectedDataPoints int64, errorMessage string) {
	var envelope struct {
		PartialSuccess struct {
			RejectedDataPoints int64  `json:"rejectedDataPoints"`
			ErrorMessage       string `json:"errorMessage"`
		} `json:"partialSuccess"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, ""
	}
	return envelope.PartialSuccess.RejectedDataPoints, envelope.PartialSuccess.ErrorMessage
}

// decodeOTLPProtoResponse extracts partial_success fields from a binary
// ExportMetricsServiceResponse protobuf body. Uses hand-rolled protowire
// scanning to avoid adding the OTel proto module as a dependency.
//
// Proto layout (simplified):
//   ExportMetricsServiceResponse {
//     1: ExportMetricsPartialSuccess partial_success {
//       1: int64  rejected_data_points
//       2: string error_message
//     }
//   }
func decodeOTLPProtoResponse(body []byte) (rejectedDataPoints int64, errorMessage string) {
	b := body
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return 0, ""
		}
		b = b[n:]

		if num == 1 && typ == protowire.BytesType {
			// partial_success message
			inner, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return 0, ""
			}
			b = b[n:]
			rejectedDataPoints, errorMessage = decodePartialSuccess(inner)
		} else {
			// skip unknown field
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return 0, ""
			}
			b = b[n:]
		}
	}
	return rejectedDataPoints, errorMessage
}

func decodePartialSuccess(b []byte) (rejectedDataPoints int64, errorMessage string) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return 0, ""
		}
		b = b[n:]

		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return 0, ""
			}
			b = b[n:]
			rejectedDataPoints = int64(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return 0, ""
			}
			b = b[n:]
			errorMessage = string(v)
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return 0, ""
			}
			b = b[n:]
		}
	}
	return rejectedDataPoints, errorMessage
}
