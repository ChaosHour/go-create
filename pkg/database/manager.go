// Package database provides MySQL database management operations including
// user creation, role management, privilege grants, and transaction handling.
// It supports both MySQL 5.7 and 8.0+ with automatic version detection.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ChaosHour/go-create/pkg/auth"
	"github.com/fatih/color"
)

// allowedPrivileges is the set of MySQL privilege keywords accepted by -g.
// This allowlist prevents arbitrary SQL injection via the grants flag.
var allowedPrivileges = map[string]struct{}{
	"all":                     {},
	"all privileges":          {},
	"select":                  {},
	"insert":                  {},
	"update":                  {},
	"delete":                  {},
	"create":                  {},
	"drop":                    {},
	"reload":                  {},
	"shutdown":                {},
	"process":                 {},
	"file":                    {},
	"references":              {},
	"index":                   {},
	"alter":                   {},
	"show databases":          {},
	"super":                   {},
	"create temporary tables": {},
	"lock tables":             {},
	"execute":                 {},
	"replication slave":       {},
	"replication client":      {},
	"create view":             {},
	"show view":               {},
	"create routine":          {},
	"alter routine":           {},
	"create user":             {},
	"event":                   {},
	"trigger":                 {},
	"create tablespace":       {},
	"usage":                   {},
}

// validateGrants checks each comma-separated privilege in the grants string
// against the known MySQL privilege allowlist and returns an error if any
// unrecognised value is found.
func validateGrants(grants string) error {
	for _, g := range strings.Split(grants, ",") {
		priv := strings.ToLower(strings.TrimSpace(g))
		if _, ok := allowedPrivileges[priv]; !ok {
			return fmt.Errorf("unrecognised privilege %q: must be a valid MySQL privilege keyword (e.g. select, insert, update, delete)", g)
		}
	}
	return nil
}

// Color formatters for consistent output
var (
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc() // Add the missing red color
)

// Manager handles database operations for MySQL user and role management.
// It provides transaction support, password policy enforcement, and
// MySQL version detection for compatibility across 5.7 and 8.0+.
type Manager struct {
	DB             *sql.DB
	Tx             *sql.Tx
	Logger         *log.Logger
	PasswordPolicy auth.PasswordPolicy
	AuthPlugin     string // Optional override for authentication plugin
	Host           string // Add Host field for connection details
	Username       string // Add Username field for connection details
	Password       string // Add Password field for connection details
}

// NewManager creates a new database manager with the specified connection and credentials.
// It initializes the password policy with default settings requiring strong passwords
// for new user creation (30+ chars, mixed case, digits, special chars).
func NewManager(db *sql.DB, host, username, password string) *Manager {
	return &Manager{
		DB:             db,
		Logger:         log.New(os.Stdout, "", log.LstdFlags),
		PasswordPolicy: auth.DefaultPasswordPolicy(),
		Host:           host,
		Username:       username,
		Password:       password,
	}
}

// BeginTx starts a transaction
func (dm *Manager) BeginTx(ctx context.Context) error {
	var err error
	dm.Tx, err = dm.DB.BeginTx(ctx, nil)
	return err
}

// CommitTx commits the current transaction
func (dm *Manager) CommitTx() error {
	return dm.Tx.Commit()
}

// RollbackTx rolls back the current transaction
func (dm *Manager) RollbackTx() error {
	return dm.Tx.Rollback()
}

// execer returns the active transaction if one is open, otherwise the plain DB.
// All mutation operations should use this so they participate in the transaction.
func (dm *Manager) execer() interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
} {
	if dm.Tx != nil {
		return dm.Tx
	}
	return dm.DB
}

// GetMySQLVersion returns MySQL version (57 for 5.7, 80 for 8.0+)
func (dm *Manager) GetMySQLVersion() (int, error) {
	var version string
	err := dm.DB.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(version, "5.7") {
		return 57, nil
	}
	return 80, nil // Assume 8.0+ for anything else
}

// ShowRoleGrants displays grants for a role
func (dm *Manager) ShowRoleGrants(role string) error {
	rows, err := dm.DB.Query("SHOW GRANTS FOR `" + role + "`")
	if err != nil {
		// Handle case where role doesn't exist
		if strings.Contains(err.Error(), "Error 1141") { // MySQL error 1141 is "unknown user"
			dm.Logger.Printf("%s Role '%s' not found", yellow("[!]"), role)
			return nil
		}
		return fmt.Errorf("showing grants: %w", err)
	}
	defer rows.Close()

	var foundGrants bool
	dm.Logger.Printf("%s Grants for role %s:", green("[+]"), role)
	for rows.Next() {
		foundGrants = true
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return fmt.Errorf("scanning grants: %w", err)
		}
		dm.Logger.Printf("    %s", grant)
	}

	if !foundGrants {
		dm.Logger.Printf("    No specific grants found for this role")
	}

	return rows.Err()
}

// ShowUserGrants displays grants for a user
func (dm *Manager) ShowUserGrants(username string) error {
	rows, err := dm.DB.Query("SHOW GRANTS FOR `" + username + "`")
	if err != nil {
		return fmt.Errorf("showing user grants: %w", err)
	}
	defer rows.Close()

	dm.Logger.Printf("%s Grants for user %s:", green("[+]"), username)
	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return fmt.Errorf("scanning grants: %w", err)
		}
		dm.Logger.Printf("    %s", grant)
	}
	return rows.Err()
}

// CheckUserExists checks if a MySQL user exists and returns their host
func (dm *Manager) CheckUserExists(username string) (bool, string, error) {
	var count int
	var host string = "%"

	err := dm.DB.QueryRow("SELECT COUNT(*) FROM mysql.user WHERE User=?", username).Scan(&count)
	if err != nil {
		return false, host, err
	}

	if count > 0 {
		// Try to get a specific host if one exists
		err = dm.DB.QueryRow("SELECT Host FROM mysql.user WHERE User=? AND Host != '%' LIMIT 1", username).Scan(&host)
		if err == sql.ErrNoRows {
			// No specific host found, use default '%'
			host = "%"
			err = nil
		}
	}

	return count > 0, host, err
}

// CreateUser creates a new MySQL user
func (dm *Manager) CreateUser(username, password string) (string, error) {
	// Validate password against policy - this only applies to new user creation.
	// Discard the shell-warning string; main.go already printed it during pre-validation.
	if _, err := auth.ValidatePassword(password, dm.PasswordPolicy); err != nil {
		return "", fmt.Errorf("new user password policy violation: %w", err)
	}

	exists, host, err := dm.CheckUserExists(username)
	if err != nil {
		return "", fmt.Errorf("checking user existence: %w", err)
	}

	if exists {
		dm.Logger.Printf("%s User %s@%s already exists", yellow("[!]"), username, host)
		return host, nil
	}

	// Get MySQL version to determine which authentication plugin to use
	version, err := dm.GetMySQLVersion()
	if err != nil {
		return "", fmt.Errorf("checking MySQL version: %w", err)
	}

	var authPlugin string

	// Use forced plugin if specified, otherwise select based on version
	if dm.AuthPlugin != "" {
		authPlugin = dm.AuthPlugin
	} else if version < 80 {
		authPlugin = "mysql_native_password"
	} else {
		authPlugin = "caching_sha2_password"
	}

	// tryCreate attempts to create the user and returns nil on success.
	tryCreate := func(query string) error {
		_, e := dm.execer().Exec(query)
		return e
	}

	escPassword := strings.Replace(password, "'", "''", -1)

	var createErr error
	if dm.AuthPlugin == "" {
		// Attempt 1: default auth (no explicit plugin)
		createErr = tryCreate(fmt.Sprintf(
			"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, escPassword))
	}

	if dm.AuthPlugin != "" || createErr != nil {
		// Attempt 2: explicit auth plugin
		createErr = tryCreate(fmt.Sprintf(
			"CREATE USER '%s'@'%%' IDENTIFIED WITH %s BY '%s'", username, authPlugin, escPassword))
	}

	if createErr != nil {
		// Attempt 3: final fallback — plain IDENTIFIED BY
		createErr = tryCreate(fmt.Sprintf(
			"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, escPassword))
	}

	if createErr != nil {
		dm.Logger.Printf("%s All user creation methods failed. Try using -use-sql-file flag for complex passwords", red("✘"))
		return "", fmt.Errorf("creating user: %w", createErr)
	}

	// Verify which plugin was actually used and correct if necessary
	var usedPlugin string
	if err = dm.DB.QueryRow("SELECT plugin FROM mysql.user WHERE User = ? AND Host = '%'", username).Scan(&usedPlugin); err != nil {
		dm.Logger.Printf("%s Could not verify authentication plugin: %v", yellow("[!]"), err)
	} else if usedPlugin != authPlugin {
		dm.Logger.Printf("%s Incorrect plugin used (%s vs %s), attempting to correct...",
			yellow("[!]"), usedPlugin, authPlugin)
		alterQuery := fmt.Sprintf(
			"ALTER USER '%s'@'%%' IDENTIFIED WITH %s BY '%s'",
			username, authPlugin, escPassword)
		if _, err = dm.execer().Exec(alterQuery); err != nil {
			dm.Logger.Printf("%s Failed to update authentication plugin: %v", yellow("[!]"), err)
		} else {
			dm.Logger.Printf("%s Successfully updated authentication plugin to %s", green("[+]"), authPlugin)
		}
	}

	dm.Logger.Printf("%s Created user: %s@%%", green("[+]"), username)
	return "%", nil
}

// CreateRole creates a new MySQL role
func (dm *Manager) CreateRole(role string) error {
	version, err := dm.GetMySQLVersion()
	if err != nil {
		return fmt.Errorf("checking MySQL version: %w", err)
	}

	if version < 80 {
		dm.Logger.Printf("%s Roles are not supported in MySQL 5.7, skipping role creation for: %s", yellow("[!]"), role)
		return nil
	}

	exists, _, err := dm.CheckUserExists(role)
	if err != nil {
		return fmt.Errorf("checking role existence: %w", err)
	}

	if exists {
		dm.Logger.Printf("%s Role %s already exists", yellow("[!]"), role)
		return nil
	}

	// Create role directly since MySQL doesn't support prepared statements for CREATE ROLE
	_, err = dm.execer().Exec(fmt.Sprintf("CREATE ROLE `%s`", role))
	if err != nil {
		return fmt.Errorf("creating role: %w", err)
	}

	dm.Logger.Printf("%s Created role: %s", green("[+]"), role)
	return nil
}

// GrantPrivileges grants privileges to a role
func (dm *Manager) GrantPrivileges(role, dbName, grants string) error {
	if err := validateGrants(grants); err != nil {
		return err
	}
	var query string
	if dbName == "*.*" {
		query = fmt.Sprintf("GRANT %s ON *.* TO `%s`", grants, role)
	} else {
		query = fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role)
	}
	_, err := dm.execer().Exec(query)
	if err != nil {
		return fmt.Errorf("granting privileges: %w", err)
	}
	dm.Logger.Printf("%s Granted privileges to role: %s", green("[+]"), role)
	return nil
}

// GetUserHost returns the host for a given user
func (dm *Manager) GetUserHost(username string) (string, error) {
	userHost := "%"
	var count int
	err := dm.DB.QueryRow("SELECT COUNT(*) FROM mysql.user WHERE User = ?", username).Scan(&count)
	if err != nil {
		return userHost, fmt.Errorf("checking user existence: %w", err)
	}

	if count > 0 {
		// Check if there's a specific host other than '%'
		err = dm.DB.QueryRow("SELECT Host FROM mysql.user WHERE User = ? AND Host != '%' LIMIT 1", username).Scan(&userHost)
		if err == sql.ErrNoRows {
			// No specific host found, use default '%'
			userHost = "%"
			err = nil
		}
	}

	return userHost, err
}

// GrantRoles grants roles to a user
func (dm *Manager) GrantRoles(username, role string, isGCP bool) error {
	version, err := dm.GetMySQLVersion()
	if err != nil {
		return fmt.Errorf("checking MySQL version: %w", err)
	}

	if version < 80 {
		dm.Logger.Printf("%s Roles are not supported in MySQL 5.7, skipping role grant for user: %s", yellow("[!]"), username)
		return nil
	}

	// Get the user's host before any operations
	userHost, err := dm.GetUserHost(username)
	if err != nil {
		return fmt.Errorf("getting user host: %w", err)
	}

	// grant privileges to the role
	_, err = dm.execer().Exec(fmt.Sprintf("GRANT `%s` TO `%s`", role, username))
	if err != nil {
		return fmt.Errorf("granting role: %w", err)
	}
	dm.Logger.Printf("%s Granted role to user: %s", green("[+]"), username)

	// If isGCP flag is set, revoke cloudsqlsuperuser role
	if isGCP {
		revokeQuery := fmt.Sprintf("REVOKE IF EXISTS 'cloudsqlsuperuser' FROM '%s'@'%s'", username, userHost)
		_, err = dm.execer().Exec(revokeQuery)
		if err != nil {
			dm.Logger.Printf("%s Warning: Failed to revoke cloudsqlsuperuser from %s@%s: %v", yellow("[!]"), username, userHost, err)
		} else {
			dm.Logger.Printf("%s Revoked cloudsqlsuperuser role from user: %s@%s", green("[+]"), username, userHost)
		}
	}
	return nil
}

// GrantPrivilegesToUser grants privileges to a user
func (dm *Manager) GrantPrivilegesToUser(username, dbName, grants string) error {
	if err := validateGrants(grants); err != nil {
		return err
	}
	var query string
	if dbName == "*.*" {
		// Get existing global privileges
		rows, err := dm.DB.Query(fmt.Sprintf("SHOW GRANTS FOR '%s'@'%%'", username))
		if err != nil {
			return fmt.Errorf("fetching existing grants: %w", err)
		}
		defer rows.Close()

		var existingGrants string
		for rows.Next() {
			var grant string
			if err := rows.Scan(&grant); err != nil {
				return fmt.Errorf("scanning grants: %w", err)
			}
			if strings.Contains(grant, "ON *.*") {
				existingGrants = strings.TrimSpace(strings.Split(strings.Split(grant, "GRANT")[1], "ON")[0])
				break
			}
		}

		// Combine existing and new privileges
		allGrants := grants
		if existingGrants != "" && existingGrants != "USAGE" {
			allGrants = existingGrants + "," + grants
		}

		query = fmt.Sprintf("GRANT %s ON *.* TO `%s`@'%%'", allGrants, username)
	} else {
		query = fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, username)
	}

	_, err := dm.execer().Exec(query)
	if err != nil {
		return fmt.Errorf("granting privileges: %w", err)
	}
	dm.Logger.Printf("%s Granted privileges to user: %s", green("[+]"), username)
	return nil
}

// SetDefaultRole sets the default role for a user
func (dm *Manager) SetDefaultRole(username, role string) error {
	version, err := dm.GetMySQLVersion()
	if err != nil {
		return fmt.Errorf("checking MySQL version: %w", err)
	}

	if version < 80 {
		dm.Logger.Printf("%s Roles are not supported in MySQL 5.7, skipping default role for user: %s", yellow("[!]"), username)
		return nil
	}

	_, err = dm.execer().Exec(fmt.Sprintf("ALTER USER `%s` DEFAULT ROLE `%s`", username, role))
	if err != nil {
		return fmt.Errorf("setting default role: %w", err)
	}
	dm.Logger.Printf("%s Set default role for user: %s", green("[+]"), role)
	return nil
}
