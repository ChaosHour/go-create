package database

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
)

// SQLFileExecutor handles creating and executing SQL files to avoid shell escaping issues
type SQLFileExecutor struct {
	Host     string
	User     string
	Password string
	Logger   *log.Logger
}

// NewSQLFileExecutor creates a new executor
func NewSQLFileExecutor(host, user, password string, logger *log.Logger) *SQLFileExecutor {
	return &SQLFileExecutor{
		Host:     host,
		User:     user,
		Password: password,
		Logger:   logger,
	}
}

// ExecuteUserCreation creates a SQL file with the user creation commands and executes it
func (e *SQLFileExecutor) ExecuteUserCreation(username, password, authPlugin string, roles []string, dbName, grants string) error {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	// Create a secure temporary directory
	tempDir, err := os.MkdirTemp("", "go-create-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	// Ensure cleanup on exit
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			e.Logger.Printf("%s Warning: Failed to remove temp directory %s: %v",
				yellow("[!]"), tempDir, err)
		}
	}()

	// Create a secure temporary SQL file
	sqlFile, err := os.CreateTemp(tempDir, fmt.Sprintf("create-user-%s-*.sql", username))
	if err != nil {
		return fmt.Errorf("creating temp SQL file: %w", err)
	}
	filename := sqlFile.Name()
	sqlFile.Close() // Close it so we can write to it later

	// Create SQL statements with proper role-based access control
	var sqlCommands []string

	// For MySQL 8.0+ with roles:
	if len(roles) > 0 {
		// 1. First create/ensure roles exist
		for _, role := range roles {
			sqlCommands = append(sqlCommands, fmt.Sprintf(
				"CREATE ROLE IF NOT EXISTS `%s`;",
				role))
		}

		// 2. Grant privileges TO ROLES (not users) when roles are specified
		if dbName != "" && grants != "" {
			for _, role := range roles {
				if dbName == "*.*" {
					sqlCommands = append(sqlCommands, fmt.Sprintf(
						"GRANT %s ON *.* TO `%s`;",
						grants, role))
				} else {
					sqlCommands = append(sqlCommands, fmt.Sprintf(
						"GRANT %s ON `%s`.* TO `%s`;",
						grants, dbName, role))
				}
			}
		}
	}

	// 3. Create user with specified authentication method - with proper escaping
	if authPlugin != "" {
		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED WITH %s BY %s;",
			username, authPlugin, safeEncodeMySQLPassword(password)))
	} else {
		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY %s;",
			username, safeEncodeMySQLPassword(password)))
	}

	// 4. Grant roles to user (if roles are specified)
	for _, role := range roles {
		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"GRANT `%s` TO '%s'@'%%';",
			role, username))
	}

	// 5. For MySQL 5.7 or when no roles are specified:
	// Grant privileges directly to the user ONLY when NO roles are specified
	if dbName != "" && grants != "" && len(roles) == 0 {
		// Fix: Only print *.* for global grants, not *.*.*
		if dbName == "*.*" {
			e.Logger.Printf("%s Adding direct grants to user (no roles specified): GRANT %s ON *.* TO '%s'@'%%'",
				yellow("[!]"), grants, username)
		} else {
			e.Logger.Printf("%s Adding direct grants to user (no roles specified): GRANT %s ON `%s`.* TO '%s'@'%%'",
				yellow("[!]"), grants, dbName, username)
		}
		if dbName == "*.*" {
			sqlCommands = append(sqlCommands, fmt.Sprintf(
				"GRANT %s ON *.* TO '%s'@'%%';",
				grants, username))
		} else {
			sqlCommands = append(sqlCommands, fmt.Sprintf(
				"GRANT %s ON `%s`.* TO '%s'@'%%';",
				grants, dbName, username))
		}
		e.Logger.Printf("%s Adding direct grants to user (no roles specified)", yellow("[!]"))
	} else {
		e.Logger.Printf("%s Not adding direct grants: dbName='%s', grants='%s', roleCount=%d",
			yellow("[!]"), dbName, grants, len(roles))
	}

	// 6. Set default roles (if roles are specified)
	if len(roles) > 0 {
		// Create comma-separated list of roles for SET DEFAULT ROLE
		rolesList := make([]string, len(roles))
		for i, r := range roles {
			rolesList[i] = "`" + r + "`"
		}
		rolesStr := strings.Join(rolesList, ", ")

		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"SET DEFAULT ROLE %s TO '%s'@'%%';",
			rolesStr, username))
	}

	// Combine all SQL statements
	sqlContent := strings.Join(sqlCommands, "\n")

	// Write SQL to file
	if err := os.WriteFile(filename, []byte(sqlContent), 0600); err != nil {
		return fmt.Errorf("writing SQL file: %w", err)
	}

	// Log SQL file creation without exposing passwords
	e.Logger.Printf("%s SQL file created with user creation commands (credentials masked for security)", green("[+]"))
	// Show structure without passwords for debugging if needed
	for _, line := range strings.Split(sqlContent, "\n") {
		if strings.Contains(line, "BY") && (strings.Contains(line, "IDENTIFIED") || strings.Contains(line, "password=")) {
			// Mask the password portion
			e.Logger.Printf("    %s", maskPasswordInSQL(line))
		} else {
			e.Logger.Printf("    %s", line)
		}
	}

	// Note: File cleanup is handled by the deferred RemoveAll at the function start

	e.Logger.Printf("%s Created SQL file for user creation: %s",
		green("[+]"), filename)

	// Try multiple connection methods if needed
	var output []byte

	// Attempt 1: Using environment variable
	e.Logger.Printf("%s Trying connection method 1: Using environment variable...", yellow("[!]"))
	cmd := exec.Command("mysql", "-h", e.Host, "-u", e.User, "-e", fmt.Sprintf("source %s", filename))
	cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", e.Password))
	output, err = cmd.CombinedOutput()
	if err == nil {
		e.Logger.Printf("%s Method 1 successful", green("[+]"))
		return nil
	}
	e.Logger.Printf("%s Method 1 failed: %v", yellow("[!]"), err)

	// Attempt 2: Using password file
	e.Logger.Printf("%s Trying connection method 2: Using password file...", yellow("[!]"))
	pwdFile, err := os.CreateTemp(tempDir, "mysql-pwd-*.txt")
	if err == nil {
		pwdFilename := pwdFile.Name()
		if _, err := pwdFile.Write([]byte(e.Password)); err != nil {
			e.Logger.Printf("%s Failed to write password file: %v", yellow("[!]"), err)
		}
		pwdFile.Close()
		// No need for explicit defer os.Remove - parent tempDir cleanup will handle it

		cmd = exec.Command("mysql", "-h", e.Host, "-u", e.User,
			fmt.Sprintf("--password-file=%s", pwdFilename), "-e", fmt.Sprintf("source %s", filename))
		output, err = cmd.CombinedOutput()
		if err == nil {
			e.Logger.Printf("%s Method 2 successful", green("[+]"))
			return nil
		}
		e.Logger.Printf("%s Method 2 failed: %v", yellow("[!]"), err)
	}

	// Attempt 3: Create a temporary configuration file
	e.Logger.Printf("%s Trying connection method 3: Using temporary my.cnf...", yellow("[!]"))
	cnfFile, err := os.CreateTemp(tempDir, "my-temp-*.cnf")
	if err == nil {
		cnfFilename := cnfFile.Name()
		cnfContent := fmt.Sprintf("[client]\nuser=%s\npassword=\"%s\"\nhost=%s\n",
			e.User, escapeCnfPassword(e.Password), e.Host)
		if _, err := cnfFile.Write([]byte(cnfContent)); err != nil {
			e.Logger.Printf("%s Failed to write config file: %v", yellow("[!]"), err)
		}
		cnfFile.Close()
		// No need for explicit defer os.Remove - parent tempDir cleanup will handle it

		cmd = exec.Command("mysql", fmt.Sprintf("--defaults-file=%s", cnfFilename),
			"-e", fmt.Sprintf("source %s", filename))
		output, err = cmd.CombinedOutput()
		if err == nil {
			e.Logger.Printf("%s Method 3 successful", green("[+]"))
			return nil
		}
		e.Logger.Printf("%s Method 3 failed: %v", yellow("[!]"), err)
	}

	// Return the original error if all methods fail
	return fmt.Errorf("all connection methods failed, last error: %w\nOutput: %s",
		err, string(output))
}

// safeEncodeMySQLPassword provides extra-safe encoding of passwords for MySQL
// This handles complex special characters by properly escaping them for SQL
func safeEncodeMySQLPassword(password string) string {
	// Instead of using UNHEX which is causing errors, use MySQL's standard string escaping

	// First, escape any single quotes by doubling them (SQL standard)
	escaped := strings.Replace(password, "'", "''", -1)

	// For backslashes, we need to double them since they're escape characters in MySQL
	escaped = strings.Replace(escaped, "\\", "\\\\", -1)

	// Some specific characters might need additional escaping
	escaped = strings.Replace(escaped, "\r", "\\r", -1)
	escaped = strings.Replace(escaped, "\n", "\\n", -1)
	escaped = strings.Replace(escaped, "\t", "\\t", -1)

	// Log the simpler escaping approach
	log.Printf("%s Using standard SQL escaping for password", yellow("[!]"))

	// Return the password as a normal SQL string literal
	return fmt.Sprintf("'%s'", escaped)
}

// Helper function to escape passwords in cnf files
func escapeCnfPassword(s string) string {
	// Escape backslashes first, then double quotes
	s = strings.Replace(s, "\\", "\\\\", -1)
	return strings.Replace(s, "\"", "\\\"", -1)
}

// maskPasswordInSQL masks password values in SQL statements for logging
func maskPasswordInSQL(sql string) string {
	// Pattern: IDENTIFIED BY 'password' or password="value"
	if idx := strings.Index(sql, "BY '"); idx != -1 {
		// Find the closing quote
		start := idx + 4
		if end := strings.Index(sql[start:], "'"); end != -1 {
			return sql[:start] + "****MASKED****" + sql[start+end:]
		}
	}
	if idx := strings.Index(sql, "password=\""); idx != -1 {
		start := idx + 10
		if end := strings.Index(sql[start:], "\""); end != -1 {
			return sql[:start] + "****MASKED****" + sql[start+end:]
		}
	}
	return sql
}
