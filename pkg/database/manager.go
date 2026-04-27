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
	// Add debug output to confirm policy is set
	policy := auth.DefaultPasswordPolicy()
	log.Printf("Initializing with password policy: min length %d chars", policy.MinLength)

	return &Manager{
		DB:             db,
		Logger:         log.New(os.Stdout, "", log.LstdFlags),
		PasswordPolicy: policy,
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
	// Add debug output to verify policy is being applied ONLY for new user creation
	dm.Logger.Printf("%s Validating new user password against policy (min length: %d)...",
		yellow("[!]"), dm.PasswordPolicy.MinLength)

	// Validate password against policy - this only applies to new user creation
	if err := auth.ValidatePassword(password, dm.PasswordPolicy); err != nil {
		dm.Logger.Printf("%s New user password policy violation: %v", yellow("[!]"), err)
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

	// Log the actual MySQL version for debugging
	var versionStr string
	if err := dm.DB.QueryRow("SELECT @@version").Scan(&versionStr); err != nil {
		dm.Logger.Printf("%s Could not query MySQL version: %v", yellow("[!]"), err)
	} else {
		dm.Logger.Printf("%s MySQL server version: %s (parsed as: %d)", yellow("[!]"), versionStr, version)
	}

	var authPlugin string

	// Use forced plugin if specified, otherwise select based on version
	if dm.AuthPlugin != "" {
		authPlugin = dm.AuthPlugin
		dm.Logger.Printf("%s Using forced authentication plugin: %s", yellow("[!]"), authPlugin)
	} else if version < 80 {
		authPlugin = "mysql_native_password"
		dm.Logger.Printf("%s Using mysql_native_password for MySQL 5.7", yellow("[!]"))
	} else {
		authPlugin = "caching_sha2_password"
		dm.Logger.Printf("%s Using caching_sha2_password for MySQL 8.0+", yellow("[!]"))
	}

	// First try using prepared statement approach
	if dm.AuthPlugin == "" {
		// For default authentication without specifying plugin
		createQuery := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
			username, strings.Replace(password, "'", "''", -1))
		_, err = dm.DB.Exec(createQuery)
		if err == nil {
			goto userCreated
		}

		dm.Logger.Printf("%s First user creation attempt failed, trying with explicit auth plugin", yellow("[!]"))
	}

	// Try with explicit auth plugin
	if dm.AuthPlugin != "" || err != nil {
		escPassword := strings.Replace(password, "'", "''", -1) // Basic SQL string escaping
		createQuery := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED WITH %s BY '%s'",
			username, authPlugin, escPassword)

		dm.Logger.Printf("%s Using direct SQL with escaping: %s", yellow("[!]"),
			fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED WITH %s BY [PASSWORD]", username, authPlugin))

		_, err = dm.DB.Exec(createQuery)
		if err == nil {
			goto userCreated
		}
	}

	// Final fallback - don't use parameterized queries, they don't work for CREATE USER
	if err != nil {
		dm.Logger.Printf("%s Previous methods failed, trying final direct SQL approach", yellow("[!]"))

		// Directly escape password and use it in SQL
		escPassword := strings.Replace(password, "'", "''", -1)
		createQuery := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
			username, escPassword)

		_, err = dm.DB.Exec(createQuery)

		if err != nil {
			dm.Logger.Printf("%s All user creation methods failed. Try using -use-sql-file flag for complex passwords", red("✘"))
			return "", fmt.Errorf("creating user: %w", err)
		}
	}

userCreated:
	// Verify which plugin was actually used
	var usedPlugin string
	err = dm.DB.QueryRow("SELECT plugin FROM mysql.user WHERE User = ? AND Host = '%'", username).Scan(&usedPlugin)
	if err != nil {
		dm.Logger.Printf("%s Could not verify authentication plugin: %v", yellow("[!]"), err)
	} else {
		dm.Logger.Printf("%s User created with authentication plugin: %s", green("[+]"), usedPlugin)

		// If the wrong plugin was used, try to alter the user
		if usedPlugin != authPlugin {
			dm.Logger.Printf("%s Incorrect plugin used (%s vs %s), attempting to correct...",
				yellow("[!]"), usedPlugin, authPlugin)

			alterQuery := fmt.Sprintf(
				"ALTER USER '%s'@'%%' IDENTIFIED WITH %s BY '%s'",
				username, authPlugin, password)

			_, err = dm.DB.Exec(alterQuery)
			if err != nil {
				dm.Logger.Printf("%s Failed to update authentication plugin: %v", yellow("[!]"), err)
			} else {
				dm.Logger.Printf("%s Successfully updated authentication plugin to %s", green("[+]"), authPlugin)
			}
		}
	}

	dm.Logger.Printf("%s Created user: %s@%% with strong password", green("[+]"), username)
	dm.Logger.Printf("%s NOTE: Complex passwords with special characters may need to be escaped when used in the MySQL CLI", yellow("[!]"))
	dm.Logger.Printf("%s For complex passwords, consider using the -use-sql-file flag", yellow("[!]"))
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
	_, err = dm.DB.Exec(fmt.Sprintf("CREATE ROLE `%s`", role))
	if err != nil {
		return fmt.Errorf("creating role: %w", err)
	}

	dm.Logger.Printf("%s Created role: %s", green("[+]"), role)
	return nil
}

// GrantPrivileges grants privileges to a role
func (dm *Manager) GrantPrivileges(role, dbName, grants string) error {
	var query string
	if dbName == "*.*" {
		query = fmt.Sprintf("GRANT %s ON *.* TO `%s`", grants, role)
	} else {
		query = fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role)
	}
	_, err := dm.DB.Exec(query)
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
	_, err = dm.DB.Exec(fmt.Sprintf("GRANT `%s` TO `%s`", role, username))
	if err != nil {
		return fmt.Errorf("granting role: %w", err)
	}
	dm.Logger.Printf("%s Granted role to user: %s", green("[+]"), username)

	// If isGCP flag is set, revoke cloudsqlsuperuser role
	if isGCP {
		revokeQuery := fmt.Sprintf("REVOKE IF EXISTS 'cloudsqlsuperuser' FROM '%s'@'%s'", username, userHost)
		_, err = dm.DB.Exec(revokeQuery)
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

	_, err := dm.DB.Exec(query)
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

	_, err = dm.DB.Exec(fmt.Sprintf("ALTER USER `%s` DEFAULT ROLE `%s`", username, role))
	if err != nil {
		return fmt.Errorf("setting default role: %w", err)
	}
	dm.Logger.Printf("%s Set default role for user: %s", green("[+]"), role)
	return nil
}
