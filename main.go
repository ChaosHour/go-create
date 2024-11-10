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

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	source         = flag.String("s", "", "Source Host")
	username       = flag.String("u", "", "User")
	password       = flag.String("p", "", "Password")
	grants         = flag.String("g", "", "Comma-separated list of grants to create")
	dbName         = flag.String("db", "", "Database name")
	role           = flag.String("r", "", "Comma-separated list of roles to create")
	help           = flag.Bool("h", false, "Print help")
	showGrants     = flag.Bool("show", false, "Show grants for specified role")
	showUserGrants = flag.Bool("show-user", false, "Show grants for specified user")
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

// Initialize flags with validation
func init() {
	flag.Parse()

	if *source != "" && !strings.Contains(*source, ":") {
		*source = *source + ":3306"
	}

	// Validate required flags when not using help
	if !*help && (*source == "" || (*username == "" && *role == "")) {
		log.Fatal("Required flags missing. Use -h for help")
	}
}

// read the ~/.my.cnf file to get the database credentials
func readMyCnf() {
	file, err := os.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "user") {
			os.Setenv("MYSQL_USER", strings.TrimSpace(line[5:]))
		}
		if strings.HasPrefix(line, "password") {
			os.Setenv("MYSQL_PASSWORD", strings.TrimSpace(line[9:]))
		}
	}
}

func checkConnection(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func connectToDatabase() *DBManager {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/",
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		*source)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("%s Failed to connect: %v", red("✘"), err)
	}

	// Test the connection
	if err := checkConnection(db); err != nil {
		log.Fatalf("%s Failed to connect to %s: %v", red("✘"), *source, err)
	}

	fmt.Printf("%s Successfully connected to %s\n", green("✓"), *source)

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
	// grant privileges to the role
	_, err := dm.db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role))
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Granted privileges to role: %s", green("[+]"), role)
}

// create a function to create a user or users
func (dm *DBManager) createUser(username string, password string) {
	// check if the user already exists
	exists, err := dm.checkUserExists(username)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		dm.logger.Printf("%s User %s already exists", yellow("[!]"), username)
	} else {
		// create the user
		_, err := dm.db.Exec(fmt.Sprintf("CREATE USER `%s` IDENTIFIED BY '%s'", username, password))
		if err != nil {
			log.Fatal(err)
		}
		dm.logger.Printf("%s Created user: %s", green("[+]"), username)
	}
}

// create a function to grant roles to the user or users
func (dm *DBManager) grantRoles(username string, role string) {
	// grant privileges to the role
	_, err := dm.db.Exec(fmt.Sprintf("GRANT `%s` TO `%s`", role, username))
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Granted role to user: %s", green("[+]"), username)
}

// create a function to grant privileges to the user or users
func (dm *DBManager) grantPrivilegesToUser(username string, dbName string, grants string) {
	// grant privileges to the role
	_, err := dm.db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, username))
	if err != nil {
		log.Fatal(err)
	}
	dm.logger.Printf("%s Granted privileges to user: %s", green("[+]"), username)
}

// function to add SET DEFAULT ROLE to the user
func (dm *DBManager) setDefaultRole(username string, role string) {
	_, err := dm.db.Exec(fmt.Sprintf("ALTER USER `%s` DEFAULT ROLE `%s`", username, role))
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

	// read the ~/.my.cnf file to get the database credentials
	readMyCnf()

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

	if *showUserGrants && *username != "" {
		users := strings.Split(*username, ",")
		for _, u := range users {
			if err := dbManager.showUserGrants(u); err != nil {
				log.Fatalf("Failed to show user grants: %v", err)
			}
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
	if *username != "" && *password != "" {
		users := strings.Split(*username, ",")
		passwords := strings.Split(*password, ",")
		for i, u := range users {
			dbManager.createUser(u, passwords[i])
		}
	}

	// grant roles to users
	if *username != "" && *role != "" {
		users := strings.Split(*username, ",")
		roles := strings.Split(*role, ",")
		for _, u := range users {
			for _, r := range roles {
				dbManager.grantRoles(u, r)
			}
		}
	}

	// grant privileges to users
	if *username != "" && *dbName != "" && *grants != "" {
		users := strings.Split(*username, ",")
		for _, u := range users {
			dbManager.grantPrivilegesToUser(u, *dbName, *grants)
		}
	}

	// set default role for users
	if *username != "" && *role != "" {
		users := strings.Split(*username, ",")
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
