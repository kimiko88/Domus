package network

// Export internal parsing functions for unit testing
func ParseWindowsARPForTest(output []byte) []DiscoveredHost {
	return parseWindowsARP(output)
}

func ParseLinuxIPNeighForTest(output []byte) []DiscoveredHost {
	return parseLinuxIPNeigh(output)
}
