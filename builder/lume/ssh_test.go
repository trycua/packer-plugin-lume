package lume

import (
	"testing"
)

func TestCaptureIPAddress(t *testing.T) {

	re := localIPv4Regex

	// Define test cases.
	tests := []struct {
		input    string
		expected string // Expected captured IP address (if any)
		found    bool   // Indicates if a match is expected
	}{
		{
			input:    "The server is at 192.168.1.1 and is online.",
			expected: "192.168.1.1",
			found:    true,
		},
		{
			input:    "No IP here.",
			expected: "",
			found:    false,
		},
		{
			input:    "Multiple IPs: 10.0.0.1 and 172.16.0.5",
			expected: "10.0.0.1", // Only the first match will be captured
			found:    true,
		},
		{
			input:    "Edge case: 256.256.256.256",
			expected: "256.256.256.256", // Pattern matches but doesn't enforce numeric range
			found:    true,
		},
	}

	for _, test := range tests {
		// Find the first match and its submatches.
		matches := re.FindStringSubmatch(test.input)
		if test.found {
			if len(matches) < 2 {
				t.Errorf("Expected to capture an IP address in %q, but got no capture", test.input)
			} else if matches[1] != test.expected {
				t.Errorf("For input %q, expected capture %q, but got %q", test.input, test.expected, matches[1])
			}
		} else {
			if len(matches) > 0 {
				t.Errorf("For input %q, expected no match, but found %q", test.input, matches[0])
			}
		}
	}
}
