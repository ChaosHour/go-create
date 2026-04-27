package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
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

	// Build SQL statements in memory — no temp files or CLI binary needed.
	var sqlCommands []string

	// 1. Create roles first (idempotent)
	for _, role := range roles {
		sqlCommands = append(sqlCommands, fmt.Sprintf("CREATE ROLE IF NOT EXISTS `%s`", role))
	}

	// 2. Grant privileges to roles
	if len(roles) > 0 && dbName != "" && grants != "" {
		for _, role := range roles {
			if dbName == "*.*" {
				sqlCommands = append(sqlCommands, fmt.Sprintf("GRANT %s ON *.* TO `%s`", grants, role))
			} else {
				sqlCommands = append(sqlCommands, fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role))
			}
		}
	}

	// 3. Create user
	if authPlugin != "" {
		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED WITH %s BY %s",
			username, authPlugin, safeEncodeMySQLPassword(password)))
	} else {
		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY %s",
			username, safeEncodeMySQLPassword(password)))
	}

	// 4. Grant roles to user
	for _, role := range roles {
		sqlCommands = append(sqlCommands, fmt.Sprintf("GRANT `%s` TO '%s'@'%%'", role, username))
	}

	// 5. Grant privileges directly to user when no roles are specified
	if dbName != "" && grants != "" && len(roles) == 0 {
		if dbName == "*.*" {
			e.Logger.Printf("%s Adding direct grants: GRANT %s ON *.* TO '%s'@'%%'", yellow("[!]"), grants, username)
			sqlCommands = append(sqlCommands, fmt.Sprintf("GRANT %s ON *.* TO '%s'@'%%'", grants, username))
		} else {
			e.Logger.Printf("%s Adding direct grants: GRANT %s ON `%s`.* TO '%s'@'%%'", yellow("[!]"), grants, dbName, username)
			sqlCommands = append(sqlCommands, fmt.Sprintf("GRANT %s ON `%s`.* TO '%s'@'%%'", grants, dbName, username))
		}
	} else {
		e.Logger.Printf("%s Not adding direct grants: dbName='%s', grants='%s', roleCount=%d",
			yellow("[!]"), dbName, grants, len(roles))
	}

	// 6. Set default roles
	if len(roles) > 0 {
		rolesList := make([]string, len(roles))
		for i, r := range roles {
			rolesList[i] = "`" + r + "`"
		}
		sqlCommands = append(sqlCommands, fmt.Sprintf(
			"SET DEFAULT ROLE %s TO '%s'@'%%'",
			strings.Join(rolesList, ", "), username))
	}

	// Log statements with passwords masked
	e.Logger.Printf("%s Executing %d statements via Go MySQL driver:", green("[+]"), len(sqlCommands))
	for _, stmt := range sqlCommands {
		e.Logger.Printf("    %s", maskPasswordInSQL(stmt))
	}

	// Open a direct connection — no mysql CLI binary required.
	// Strip any DSN query params from host; the driver handles host:port natively.
	tcpHost := e.Host
	if idx := strings.Index(tcpHost, "?"); idx != -1 {
		tcpHost = tcpHost[:idx]
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/", e.User, e.Password, tcpHost)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("opening connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("connecting to MySQL at %s: %w", tcpHost, err)
	}

	for _, stmt := range sqlCommands {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("executing %q: %w", maskPasswordInSQL(stmt), err)
		}
	}

	e.Logger.Printf("%s User '%s' created successfully", green("[+]"), username)
	return nil
}

// safeEncodeMySQLPassword escapes a password for use in a SQL string literal.
func safeEncodeMySQLPassword(password string) string {
	escaped := strings.Replace(password, "'", "''", -1)
	escaped = strings.Replace(escaped, "\\", "\\\\", -1)
	escaped = strings.Replace(escaped, "\r", "\\r", -1)
	escaped = strings.Replace(escaped, "\n", "\\n", -1)
	escaped = strings.Replace(escaped, "\t", "\\t", -1)
	log.Printf("%s Using standard SQL escaping for password", yellow("[!]"))
	return fmt.Sprintf("'%s'", escaped)
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
