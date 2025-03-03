package auth

import (
	"fmt"
	"os"
	"strings"
)

// Credentials holds database connection details
type Credentials struct {
	User     string
	Password string
	Host     string
	Port     string
}

// ReadMyCnf reads MySQL credentials from ~/.my.cnf
func ReadMyCnf() (string, string, string) {
	var host, user, password, port string
	var inClientSection bool

	file, err := os.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		return "", "", ""
	}

	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.ToLower(strings.Trim(line, "[]"))
			inClientSection = section == "client"
			continue
		}

		if !inClientSection {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "'\"")

		switch key {
		case "user":
			user = value
		case "password":
			password = value
		case "host":
			host = value
		case "port":
			port = value
		}
	}

	// Build and return the host string if both host and port were found
	if host != "" {
		if port != "" {
			host = fmt.Sprintf("%s:%s", host, port)
		} else {
			host = fmt.Sprintf("%s:3306", host)
		}
	}

	return host, user, password
}

// BuildDSN constructs a MySQL DSN connection string
func BuildDSN(user, password, host string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/", user, password, host)
}
