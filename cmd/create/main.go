package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ChaosHour/go-create/internal/config"
	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	source         = flag.String("s", "", "Source Host to connect to")
	username       = flag.String("u", "", "Username to connect with (from .my.cnf if not specified)")
	password       = flag.String("p", "", "Password to connect with (from .my.cnf if not specified)")
	createUser     = flag.String("create-user", "", "Username to create/modify")
	createPassword = flag.String("create-pass", "", "Password for the user being created")
	grants         = flag.String("g", "", "Comma-separated list of grants to create")
	dbName         = flag.String("db", "", "Database name")
	role           = flag.String("r", "", "Comma-separated list of roles to create")
	help           = flag.Bool("h", false, "Print help")
	showGrants     = flag.Bool("show", false, "Show grants for specified role")
	showUserName   = flag.String("show-user", "", "Show grants for the specified username")
	configFile     = flag.String("config", "", "Path to configuration file")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()

//var blue = color.New(color.FgBlue).SprintFunc()

// Configuration holds database connection details
type Configuration struct {
	User     string
	Password string
	Host     string
}

// DBManager handles database operations
type DBManager struct {
	db     *sql.DB
	tx     *sql.Tx
	logger *log.Logger
}

// Add transaction methods to DBManager
func (dm *DBManager) beginTx(ctx context.Context) error {
	var err error
	dm.tx, err = dm.db.BeginTx(ctx, nil)
	return err
}

func (dm *DBManager) commitTx() error {
	return dm.tx.Commit()
}

func (dm *DBManager) rollbackTx() error {
	return dm.tx.Rollback()
}

// Add new method to DBManager
func (dm *DBManager) showRoleGrants(role string) error {
	rows, err := dm.db.Query("SHOW GRANTS FOR `" + role + "`")
	if err != nil {
		return fmt.Errorf("showing grants: %w", err)
	}
	defer rows.Close()

	dm.logger.Printf("%s Grants for role %s:", green("[+]"), role)
	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return fmt.Errorf("scanning grants: %w", err)
		}
		dm.logger.Printf("    %s", grant)
	}
	return rows.Err()
}

// Add new method to DBManager
func (dm *DBManager) showUserGrants(username string) error {
	rows, err := dm.db.Query("SHOW GRANTS FOR `" + username + "`")
	if err != nil {
		return fmt.Errorf("showing user grants: %w", err)
	}
	defer rows.Close()

	dm.logger.Printf("%s Grants for user %s:", green("[+]"), username)
	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return fmt.Errorf("scanning grants: %w", err)
		}
		dm.logger.Printf("    %s", grant)
	}
	return rows.Err()
}

// Add new method to check MySQL version
func (dm *DBManager) getMySQLVersion() (int, error) {
	var version string
	err := dm.db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(version, "5.7") {
		return 57, nil
	}
	return 80, nil // Assume 8.0+ for anything else
}

// Initialize flags with validation
func init() {
	flag.Parse()

	// Read .my.cnf first so we have credentials available for validation
	mycnfHost, mycnfUser, mycnfPwd := readMyCnf()

	// Skip .my.cnf message if command line source is provided
	if *source == "" && mycnfUser != "" {
		log.Printf("%s Using credentials from .my.cnf", green("[+]"))
	}

	// Add port to source if not specified
	if *source != "" && !strings.Contains(*source, ":") {
		*source = *source + ":3306"
	}

	// Load config to check for values
	cfg, err := config.LoadConfig(*configFile)
	if err != nil && *configFile != "" {
		log.Printf("%s Warning: Could not load config file: %v", yellow("[!]"), err)
	}

	// Skip remaining validation if showing help
	if *help {
		return
	}

	// Skip validation for show-user command
	if *showUserName != "" {
		return
	}

	// For other operations, check credentials
	hasConfigCredentials := cfg != nil && cfg.MySQL.User != "" && cfg.MySQL.Host != ""
	hasMyCnfCredentials := mycnfUser != "" && mycnfPwd != ""

	// Validate required flags if no config credentials available
	if !hasConfigCredentials && !hasMyCnfCredentials {
		noValidHost := *source == "" && mycnfHost == ""
		noValidUser := *username == "" && *role == ""

		if noValidHost || noValidUser {
			log.Fatal("Required flags missing. Use -h for help")
		}
	}
}

// read the ~/.my.cnf file to get the database credentials
func readMyCnf() (string, string, string) {
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

	// Only build and return the host string if both host and port were found
	if host != "" {
		if port != "" {
			host = fmt.Sprintf("%s:%s", host, port)
		} else {
			host = fmt.Sprintf("%s:3306", host)
		}
	}

	return host, user, password
}

func checkConnection(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func connectToDatabase() *DBManager {
	var dsn string
	var connectionInfo string

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil && *configFile != "" {
		log.Printf("%s Warning: Could not load config file: %v", yellow("[!]"), err)
	}

	// Initialize variables
	var (
		user string
		pwd  string
		host string
	)

	// Command line arguments take highest precedence
	if *source != "" {
		host = *source
		if !strings.Contains(host, ":") {
			host = fmt.Sprintf("%s:3306", host)
		}
		connectionInfo = "command line arguments"
	} else {
		// Try .my.cnf second
		mycnfHost, mycnfUser, mycnfPwd := readMyCnf()
		if mycnfHost != "" {
			host = mycnfHost
			connectionInfo = ".my.cnf"
		}
		if mycnfUser != "" {
			user = mycnfUser
		}
		if mycnfPwd != "" {
			pwd = mycnfPwd
		}

		// Config file third
		if cfg != nil {
			if cfg.MySQL.User != "" && user == "" {
				user = cfg.MySQL.User
			}
			if cfg.MySQL.Password != "" && pwd == "" {
				pwd = cfg.MySQL.Password
			}
			if cfg.MySQL.Host != "" && host == "" {
				host = cfg.MySQL.Host
				if cfg.MySQL.Port != "" {
					host = fmt.Sprintf("%s:%s", host, cfg.MySQL.Port)
				} else {
					host = fmt.Sprintf("%s:3306", host)
				}
				connectionInfo = *configFile
			}
		}
	}

	// Override user/password from command line if provided
	if *username != "" {
		user = *username
	}
	if *password != "" {
		pwd = *password
	}

	// Set default host if still not provided
	if host == "" {
		host = "localhost:3306"
		connectionInfo = "default settings"
	}

	if user == "" {
		log.Fatalf("%s No username specified. Use -u flag, config file, or .my.cnf", red("✘"))
	}

	// Log single, clear connection message
	log.Printf("%s Connecting to MySQL server at %s (using %s)",
		green("[+]"), host, connectionInfo)

	dsn = fmt.Sprintf("%s:%s@tcp(%s)/", user, pwd, host)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%s Failed to connect: %v", red("✘"), err)
	}

	// Test the connection
	if err := checkConnection(db); err != nil {
		log.Fatalf("%s Failed to connect to %s: %v", red("✘"), host, err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DBManager{db: db, logger: log.New(os.Stdout, "", log.LstdFlags)}
}

// check if a MySQL user with the specified username already exists
func (dm *DBManager) checkUserExists(username string) (bool, error) {
	var count int
	err := dm.db.QueryRow(fmt.Sprintf("SELECT count(*) FROM mysql.user WHERE User='%s'", username)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// create a function to create a role or roles
func (dm *DBManager) createRole(role string) error {
	version, err := dm.getMySQLVersion()
	if err != nil {
		return fmt.Errorf("checking MySQL version: %w", err)
	}

	if version < 80 {
		dm.logger.Printf("%s Roles are not supported in MySQL 5.7, skipping role creation for: %s", yellow("[!]"), role)
		return nil
	}

	exists, err := dm.checkUserExists(role)
	if err != nil {
		return fmt.Errorf("checking role existence: %w", err)
	}

	if exists {
		dm.logger.Printf("%s Role %s already exists", yellow("[!]"), role)
		return nil
	}

	// Create role directly since MySQL doesn't support prepared statements for CREATE ROLE
	_, err = dm.db.Exec(fmt.Sprintf("CREATE ROLE `%s`", role))
	if err != nil {
		return fmt.Errorf("creating role: %w", err)
	}

	dm.logger.Printf("%s Created role: %s", green("[+]"), role)
	return nil
}

// create a function to grant privileges to the role or roles
func (dm *DBManager) grantPrivileges(role string, dbName string, grants string) {
	var query string
	if dbName == "*.*" {
		query = fmt.Sprintf("GRANT %s ON *.* TO `%s`", grants, role)
	} else {
		query = fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role)
	}
	_, err := dm.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Granted privileges to role: %s", green("[+]"), role)
}

// create a function to create a user or users
func (dm *DBManager) createUser(username string, password string) {
	exists, err := dm.checkUserExists(username)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		dm.logger.Printf("%s User %s already exists", yellow("[!]"), username)
	} else {
		// Try MySQL 5.7 syntax first
		_, err := dm.db.Exec(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED WITH mysql_native_password BY '%s'", username, password))
		if err != nil {
			// If 5.7 syntax fails, try MySQL 8.x syntax
			_, err = dm.db.Exec(fmt.Sprintf("CREATE USER `%s` IDENTIFIED BY '%s'", username, password))
			if err != nil {
				log.Fatal(err)
			}
		}
		dm.logger.Printf("%s Created user: %s", green("[+]"), username)
	}
}

// create a function to grant roles to users
func (dm *DBManager) grantRoles(username string, role string) {
	version, err := dm.getMySQLVersion()
	if err != nil {
		log.Fatal(err)
	}

	if version < 80 {
		dm.logger.Printf("%s Roles are not supported in MySQL 5.7, skipping role grant for user: %s", yellow("[!]"), username)
		return
	}

	// grant privileges to the role
	_, err = dm.db.Exec(fmt.Sprintf("GRANT `%s` TO `%s`", role, username))
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Granted role to user: %s", green("[+]"), username)
}

// create a function to grant privileges to the user or users
func (dm *DBManager) grantPrivilegesToUser(username string, dbName string, grants string) {
	var query string
	if dbName == "*.*" {
		// Get existing global privileges
		rows, err := dm.db.Query(fmt.Sprintf("SHOW GRANTS FOR '%s'@'%%'", username))
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var existingGrants string
		for rows.Next() {
			var grant string
			if err := rows.Scan(&grant); err != nil {
				log.Fatal(err)
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

	_, err := dm.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Granted privileges to user: %s", green("[+]"), username)
}

// function to add SET DEFAULT ROLE to the user
func (dm *DBManager) setDefaultRole(username string, role string) {
	version, err := dm.getMySQLVersion()
	if err != nil {
		log.Fatal(err)
	}

	if version < 80 {
		dm.logger.Printf("%s Roles are not supported in MySQL 5.7, skipping default role for user: %s", yellow("[!]"), username)
		return
	}

	_, err = dm.db.Exec(fmt.Sprintf("ALTER USER `%s` DEFAULT ROLE `%s`", username, role))
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Set default role for user: %s", green("[+]"), username)
}

// main function
func main() {
	// print help menu if no flags are provided or -h flag is set
	if len(os.Args) == 1 || *help {
		flag.Usage()
		os.Exit(0)
	}

	// connect to the source database
	dbManager := connectToDatabase()
	defer dbManager.db.Close()

	// Add before transaction start:
	if *showGrants && *role != "" {
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			if err := dbManager.showRoleGrants(r); err != nil {
				log.Fatalf("Failed to show grants: %v", err)
			}
		}
		return
	}

	if *showUserName != "" {
		if err := dbManager.showUserGrants(*showUserName); err != nil {
			log.Fatalf("Failed to show user grants: %v", err)
		}
		return
	}

	ctx := context.Background()
	// Start transaction
	if err := dbManager.beginTx(ctx); err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	// create roles
	if *role != "" {
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			if err := dbManager.createRole(r); err != nil {
				log.Fatalf("Failed to create role: %v", err)
			}
		}
	}

	// grant privileges to roles
	if *role != "" && *dbName != "" && *grants != "" {
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			dbManager.grantPrivileges(r, *dbName, *grants)
		}
	}

	// create users
	if *createUser != "" && *createPassword != "" {
		users := strings.Split(*createUser, ",")
		passwords := strings.Split(*createPassword, ",")
		for i, u := range users {
			dbManager.createUser(u, passwords[i])
		}
	}

	// grant roles to users
	if *createUser != "" && *role != "" {
		users := strings.Split(*createUser, ",")
		roles := strings.Split(*role, ",")
		for _, u := range users {
			for _, r := range roles {
				dbManager.grantRoles(u, r)
			}
		}
	}

	// grant privileges to users
	if *createUser != "" && *dbName != "" && *grants != "" {
		users := strings.Split(*createUser, ",")
		for _, u := range users {
			dbManager.grantPrivilegesToUser(u, *dbName, *grants)
		}
	}

	// set default role for users
	if *createUser != "" && *role != "" {
		users := strings.Split(*createUser, ",")
		roles := strings.Split(*role, ",")
		for _, u := range users {
			for _, r := range roles {
				dbManager.setDefaultRole(u, r)
			}
		}
	}

	// Commit transaction
	if err := dbManager.commitTx(); err != nil {
		if rbErr := dbManager.rollbackTx(); rbErr != nil {
			log.Printf("Failed to rollback: %v", rbErr)
		}
		log.Fatalf("Failed to commit transaction: %v", err)
	}
}
