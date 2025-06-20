package auth

import (
	"fmt"
	"strings"
)

// BuildDSNWithParams creates a MySQL connection string (DSN) that is correctly formatted.
// It parses a host string that may contain a port and/or URL parameters.
func BuildDSNWithParams(username, password, hostString string) string {
	var (
		hostname string
		port     = "3306" // Default MySQL port
		params   string
	)

	// 1. Separate host/port from URL parameters
	if strings.Contains(hostString, "?") {
		parts := strings.SplitN(hostString, "?", 2)
		hostString = parts[0]
		params = parts[1]
	}

	// 2. Separate hostname from port
	if strings.Contains(hostString, ":") {
		parts := strings.SplitN(hostString, ":", 2)
		hostname = parts[0]
		port = parts[1]
	} else {
		hostname = hostString
	}

	// 3. Assemble the DSN in the format required by the driver
	// format: user:password@tcp(host:port)/?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", username, password, hostname, port)
	if params != "" {
		dsn += "?" + params
	}

	return dsn
}
