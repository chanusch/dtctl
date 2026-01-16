package settings

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// DecodedObjectID contains the decoded components of a settings object ID
type DecodedObjectID struct {
	SchemaID  string
	ScopeType string
	ScopeID   string
	UID       string
}

// DecodeObjectID decodes a Dynatrace settings object ID into its components.
//
// The objectId is a RawURLEncoding base64-encoded (URL-safe, no padding) binary structure with the format:
//
//	[8-byte magic header][4-byte version][length:uint16][string]...
//
// Where the strings are: schemaId, scopeType, scopeId, uid
//
// Example:
//
//	Input:  "vu9U3hXa3q0AAAABABRidWlsdGluOnJ1bS53ZWIubmFtZQAL..."
//	Output: SchemaID="builtin:rum.web.name", ScopeType="APPLICATION",
//	        ScopeID="5C9B9BB1B4546855", UID="e4c6742f-47f9-3b14-8348-59cbe32f7980"
func DecodeObjectID(objectID string) (*DecodedObjectID, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// The structure starts with an 8-byte magic header and 4-byte version
	// At offset 12 (0x0c), we have length-prefixed strings
	const headerSize = 12
	if len(decoded) < headerSize {
		return nil, fmt.Errorf("object ID too short (len=%d)", len(decoded))
	}

	result := &DecodedObjectID{}
	offset := headerSize

	// Read length-prefixed strings: schemaId, scopeType, scopeId, uid
	// Note: Not all objectIds have all fields (e.g., environment-scoped settings may lack scopeId/uid)
	fields := []*string{
		&result.SchemaID,
		&result.ScopeType,
		&result.ScopeID,
		&result.UID,
	}

	for _, field := range fields {
		// Try to read the next field, but don't fail if we reach the end
		value, newOffset, err := readLengthPrefixedString(decoded, offset)
		if err != nil {
			// If we can't read more fields, that's okay - some objectIds are shorter
			// Just return what we've got so far
			break
		}
		*field = value
		offset = newOffset
	}

	return result, nil
}

// readLengthPrefixedString reads a big-endian uint16 length followed by a UTF-8 string
func readLengthPrefixedString(data []byte, offset int) (string, int, error) {
	if offset+2 > len(data) {
		return "", offset, fmt.Errorf("insufficient data for length at offset %d", offset)
	}

	// Read big-endian uint16 length
	length := int(data[offset])<<8 | int(data[offset+1])
	offset += 2

	if offset+length > len(data) {
		return "", offset, fmt.Errorf("insufficient data for string of length %d at offset %d", length, offset)
	}

	value := string(data[offset : offset+length])
	offset += length

	return value, offset, nil
}

// FormattedScope returns the scope in "TYPE-ID" format (e.g., "APPLICATION-5C9B9BB1B4546855")
func (d *DecodedObjectID) FormattedScope() string {
	if d.ScopeType == "" && d.ScopeID == "" {
		return ""
	}
	return d.ScopeType + "-" + d.ScopeID
}

// DecodedVersion contains the decoded components of a settings version string
type DecodedVersion struct {
	UID          string     // The object UID
	RevisionUUID string     // The revision UUID (typically v1 UUID with timestamp)
	Timestamp    *time.Time // Decoded timestamp from v1 UUID (nil if not a v1 UUID)
}

// DecodeVersion decodes a Dynatrace settings version string into its components.
//
// The version is a RawURLEncoding base64-encoded binary structure containing:
//   - 8-byte magic header
//   - 2-byte field (purpose unknown)
//   - Length-prefixed UID string
//   - Length-prefixed revision UUID string
//   - Magic trailer
//
// The revision UUID is typically a v1 UUID containing a timestamp.
func DecodeVersion(version string) (*DecodedVersion, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(version)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Version structure: 8-byte header + 2-byte field + length-prefixed strings
	const headerSize = 10 // 8-byte magic + 2-byte field
	if len(decoded) < headerSize {
		return nil, fmt.Errorf("version too short (len=%d)", len(decoded))
	}

	result := &DecodedVersion{}
	offset := headerSize

	// Read UID
	uid, newOffset, err := readLengthPrefixedString(decoded, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to read UID: %w", err)
	}
	result.UID = uid
	offset = newOffset

	// Read revision UUID
	revisionUUID, _, err := readLengthPrefixedString(decoded, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to read revision UUID: %w", err)
	}
	result.RevisionUUID = revisionUUID

	// Try to extract timestamp from v1 UUID
	if ts, err := extractUUIDv1Timestamp(revisionUUID); err == nil {
		result.Timestamp = &ts
	}

	return result, nil
}

// extractUUIDv1Timestamp extracts the timestamp from a v1 UUID.
// UUID v1 contains a 60-bit timestamp measured in 100-nanosecond intervals
// since October 15, 1582 (the Gregorian calendar reform).
func extractUUIDv1Timestamp(uuidStr string) (time.Time, error) {
	// Remove hyphens and parse hex
	uuidStr = strings.ReplaceAll(uuidStr, "-", "")
	if len(uuidStr) != 32 {
		return time.Time{}, fmt.Errorf("invalid UUID length")
	}

	uuidBytes, err := hex.DecodeString(uuidStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid UUID hex: %w", err)
	}

	// Check version (should be 1 for time-based UUID)
	// Version is in the high nibble of byte 6
	version := (uuidBytes[6] >> 4) & 0x0F
	if version != 1 {
		return time.Time{}, fmt.Errorf("not a v1 UUID (version=%d)", version)
	}

	// Extract timestamp components:
	// time_low: bytes 0-3 (bits 0-31)
	// time_mid: bytes 4-5 (bits 32-47)
	// time_hi: bytes 6-7, lower 12 bits (bits 48-59)
	timeLow := uint64(uuidBytes[0])<<24 | uint64(uuidBytes[1])<<16 |
		uint64(uuidBytes[2])<<8 | uint64(uuidBytes[3])
	timeMid := uint64(uuidBytes[4])<<8 | uint64(uuidBytes[5])
	timeHi := uint64(uuidBytes[6]&0x0F)<<8 | uint64(uuidBytes[7])

	// Combine into 60-bit timestamp
	timestamp := timeLow | (timeMid << 32) | (timeHi << 48)

	// Convert from 100-nanosecond intervals since 1582-10-15 to Unix time
	// Difference between 1582-10-15 and 1970-01-01 in 100-ns intervals
	const gregorianToUnix = 122192928000000000

	unixNano := int64(timestamp-gregorianToUnix) * 100
	return time.Unix(0, unixNano).UTC(), nil
}
