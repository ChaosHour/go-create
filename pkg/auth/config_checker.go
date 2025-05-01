package auth

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// CheckMyCnfCredentialsForAdmin verifies that .my.cnf doesn't contain user credentials
// that are likely to lack admin privileges
func CheckMyCnfCredentialsForAdmin() {
	yellow := color.New(color.FgYellow).SprintFunc()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	myCnfPath := filepath.Join(homeDir, ".my.cnf")
	data, err := os.ReadFile(myCnfPath)
	if err != nil {
		return
	}

	content := string(data)

	// Extract username from the config file
	userLine := extractConfigValue(content, "user")
	if userLine == "" {
		return
	}

	// Check if username appears to be a non-admin user
	if strings.Contains(userLine, "admin") && !strings.Contains(userLine, "root") {
		log.Printf("%s WARNING: Your .my.cnf contains user '%s' which may not have privileges to create users",
			yellow("[!]"), userLine)
		log.Printf("%s For creating users, connect as MySQL root or a user with GRANT OPTION privileges",
			yellow("[!]"))
		log.Printf("%s Try: mysql -u root -p -h your_host", yellow("[!]"))
		log.Printf("%s Then: GRANT CREATE USER ON *.* TO 'your_admin_user'@'%%' WITH GRANT OPTION;", yellow("[!]"))
	}
}

// extractConfigValue gets a specific value from the config file content
func extractConfigValue(content, key string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	return ""
}
