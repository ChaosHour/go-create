// Package auth provides authentication and credential management utilities
// for MySQL connections. It includes DSN building, password validation,
// and secure credential handling with sanitization capabilities.
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

// SanitizeDSN removes sensitive information from a DSN for safe logging
func SanitizeDSN(dsn string) string {
	// Pattern: user:password@tcp(host:port)/
	if idx := strings.Index(dsn, ":"); idx != -1 {
		if endIdx := strings.Index(dsn[idx:], "@tcp"); endIdx != -1 {
			// Replace password portion
			return dsn[:idx+1] + "****@tcp" + dsn[idx+endIdx+4:]
		}
	}
	return dsn
}

// SanitizeError removes sensitive information from error messages
func SanitizeError(err error, password string) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	// Replace any occurrence of the password
	if password != "" && strings.Contains(errMsg, password) {
		errMsg = strings.ReplaceAll(errMsg, password, "****")
	}
	return fmt.Errorf("%s", errMsg)
}
