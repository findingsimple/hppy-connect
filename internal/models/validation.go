package models

import (
	"fmt"
	"math"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// validID matches numeric, UUID-style, or underscore-containing identifiers.
var validID = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

// ValidateID checks that an ID string contains only safe characters.
func ValidateID(name, value string) error {
	if value != "" && !validID.MatchString(value) {
		return fmt.Errorf("%s contains invalid characters", name)
	}
	return nil
}

// MaxFreeTextLength is the maximum length (in bytes) for free-text fields like
// description, comment, and entry notes. Prevents accidentally sending
// multi-megabyte strings to the API.
const MaxFreeTextLength = 100_000

// ValidateFileName checks that a filename does not contain path traversal characters.
func ValidateFileName(name string) error {
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("file name must not contain path separators")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("file name must not be a relative path reference")
	}
	return nil
}

// validMIME matches standard MIME type format: type/subtype (no parameters or trailing content).
var validMIME = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9!#$&\-^_.+]*\/[a-zA-Z0-9][a-zA-Z0-9!#$&\-^_.+]*$`)

// ValidateMIMEType checks that a MIME type string matches the standard type/subtype format.
func ValidateMIMEType(mimeType string) error {
	if !validMIME.MatchString(mimeType) {
		return fmt.Errorf("invalid MIME type format %q (expected type/subtype, e.g. image/jpeg)", mimeType)
	}
	return nil
}

// ValidateTimestamp checks that a string is a valid RFC3339 timestamp.
func ValidateTimestamp(name, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s must be a valid RFC3339 timestamp (e.g. 2026-05-01T09:00:00Z): %w", name, err)
	}
	return nil
}

// validDuration matches ISO 8601 duration format: PT[nH][nM][nS] with at least one component.
var validDuration = regexp.MustCompile(`^PT(\d+H)?(\d+M)?(\d+(\.\d+)?S)?$`)

// ValidateDuration checks that a string is a valid ISO 8601 duration (e.g. PT1H30M).
func ValidateDuration(value string) error {
	if value == "" {
		return nil
	}
	if !validDuration.MatchString(value) || value == "PT" {
		return fmt.Errorf("invalid duration %q (expected ISO 8601 format, e.g. PT1H30M)", value)
	}
	return nil
}

// ValidateFreeText checks that a free-text value does not exceed MaxFreeTextLength.
func ValidateFreeText(name, value string) error {
	if len(value) > MaxFreeTextLength {
		return fmt.Errorf("%s exceeds maximum length of %d bytes", name, MaxFreeTextLength)
	}
	return nil
}

// ValidateRatingScore checks that a rating score is a finite, non-negative number.
func ValidateRatingScore(score *float64) error {
	if score == nil {
		return nil
	}
	if math.IsNaN(*score) || math.IsInf(*score, 0) {
		return fmt.Errorf("rating score must be a finite number")
	}
	if *score < 0 {
		return fmt.Errorf("rating score must not be negative")
	}
	return nil
}

// ValidatePhotoSize checks that a photo size is positive when provided.
func ValidatePhotoSize(size *int) error {
	if size != nil && *size <= 0 {
		return fmt.Errorf("photo size must be a positive number of bytes")
	}
	return nil
}

// ValidateEmail performs basic email format validation.
func ValidateEmail(value string) error {
	_, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("invalid email address format")
	}
	return nil
}

// ValidateWebhookSubjects validates and normalises a comma-separated subjects string.
// Returns the normalised uppercase slice, or an error if any subject is invalid.
// Returns nil for empty input.
func ValidateWebhookSubjects(csv string) ([]string, error) {
	parts := SplitCSV(csv)
	if len(parts) == 0 {
		return nil, nil
	}
	result := make([]string, len(parts))
	for i, s := range parts {
		upper := strings.ToUpper(s)
		if !ValidWebhookSubjects[upper] {
			allowed := make([]string, 0, len(ValidWebhookSubjects))
			for k := range ValidWebhookSubjects {
				allowed = append(allowed, k)
			}
			sort.Strings(allowed)
			return nil, fmt.Errorf("invalid webhook subject %q — must be one of: %s", s, strings.Join(allowed, ", "))
		}
		result[i] = upper
	}
	return result, nil
}

// SplitCSV splits a comma-separated string into trimmed, non-empty values.
// Returns nil for empty input or input containing only whitespace/commas.
func SplitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ValidateWebhookURL validates a webhook URL for safety.
// Rejects non-HTTPS schemes, private/internal IPs, cloud metadata endpoints,
// embedded credentials, and IPv6 zone IDs.
func ValidateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTPS (got %q)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}

	// Reject embedded credentials — they would be stored/logged in plaintext.
	if u.User != nil {
		return fmt.Errorf("webhook URL must not contain embedded credentials")
	}

	hostname := u.Hostname()

	// Strip IPv6 zone ID before IP checks. Zone IDs (e.g. "fe80::1%%eth0") cause
	// net.ParseIP to return nil, which would bypass all private IP checks.
	if idx := strings.Index(hostname, "%"); idx != -1 {
		hostname = hostname[:idx]
	}

	// Reject well-known dangerous hostnames: cloud metadata endpoints and localhost.
	// Note: this does not defend against DNS rebinding (where a public hostname resolves
	// to a private IP at request time). Full protection would require resolving DNS at
	// validation time or using a custom http.Transport with DialContext filtering.
	blockedHosts := []string{"metadata.google.internal", "169.254.169.254", "localhost"}
	for _, h := range blockedHosts {
		if strings.EqualFold(hostname, h) {
			return fmt.Errorf("webhook URL must not point to dangerous hostnames (%s)", h)
		}
	}

	// If the hostname is a raw IP, check for private/internal ranges.
	// This covers IPv4, IPv6, and IPv4-mapped IPv6 (e.g. ::ffff:127.0.0.1).
	ip := net.ParseIP(hostname)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("webhook URL must not point to internal/private addresses")
		}
	}

	return nil
}
