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

	"github.com/ChaosHour/go-create/pkg/auth"
	"github.com/ChaosHour/go-create/pkg/config"
	"github.com/ChaosHour/go-create/pkg/database"
	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	source             = flag.String("s", "", "Source Host to connect to")
	username           = flag.String("u", "", "Username to connect with (from .my.cnf if not specified)")
	password           = flag.String("p", "", "Password to connect with (from .my.cnf if not specified)")
	createUser         = flag.String("create-user", "", "Username to create/modify")
	createPassword     = flag.String("create-pass", "", "Password for the user being created (subject to password policy)")
	grants             = flag.String("g", "", "Comma-separated list of grants to create")
	dbName             = flag.String("db", "", "Database name")
	role               = flag.String("r", "", "Comma-separated list of roles to create")
	help               = flag.Bool("h", false, "Print help")
	showGrants         = flag.Bool("show", false, "Show grants for specified role (requires -r flag)")
	showRoleName       = flag.String("show-role", "", "Show grants for the specified role name")
	showUserName       = flag.String("show-user", "", "Show grants for the specified username")
	configFile         = flag.String("config", "", "Path to configuration file")
	isGCP              = flag.Bool("gcp", false, "After granting roles to a user, automatically revoke the 'cloudsqlsuperuser' role (for GCP Cloud SQL)")
	skipPasswordPolicy = flag.Bool("skip-password-policy", false, "Skip password policy enforcement when creating new users with --create-user")
	authPlugin         = flag.String("auth-plugin", "", "Force a specific authentication plugin (mysql_native_password or caching_sha2_password)")
	useSQLFile         = flag.Bool("use-sql-file", false, "Create a temporary SQL file for executing commands (helps with complex passwords)")
	testConnection     = flag.Bool("test-connection", false, "Test connection with provided credentials")
	testUser           = flag.String("user", "", "Username for connection test (with -test-connection)")
	testPass           = flag.String("pass", "", "Password for connection test (with -test-connection)")
	testHost           = flag.String("host", "", "Host for connection test (with -test-connection)")
	debugPassword      = flag.Bool("debug-password", false, "Print detailed information about the password characters")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()

// Initialize flags with validation
func init() {
	flag.Parse()

	// Add clearer usage information about password policy
	if *help {
		fmt.Println("\nPASSWORD POLICY NOTE:")
		fmt.Println("  * The password policy (min 30 chars, mixed case, numbers, symbols)")
		fmt.Println("    ONLY applies when creating NEW USERS with -create-user and -create-pass")
		fmt.Println("  * It does NOT affect:")
		fmt.Println("    - MySQL connection credentials (from .my.cnf, config file, or -u/-p flags)")
		fmt.Println("    - Existing users' passwords")
		fmt.Println("    - Any other operations in the tool")
		fmt.Println("\n  Use -skip-password-policy to bypass these requirements when needed")
	}

	// Read .my.cnf first so we have credentials available for validation
	mycnfHost, mycnfUser, mycnfPwd := auth.ReadMyCnf()

	// Skip .my.cnf message if command line source is provided
	if *source == "" && mycnfUser != "" {
		log.Printf("%s Using credentials from .my.cnf", green("[+]"))
	}

	// Add port to source if not specified
	if *source != "" && !strings.Contains(*source, ":") {
		*source = *source + ":3306"
	}

	// Validate -gcp flag usage
	if *isGCP {
		if *role == "" {
			log.Fatal("The -gcp flag requires -r flag to specify roles")
		}
		if *createUser == "" && *username == "" {
			log.Fatal("The -gcp flag requires either --create-user flag to create a new user or -u to specify an existing user")
		}
		if *createUser != "" && *createPassword == "" {
			log.Fatal("When using --create-user with -gcp, you must also specify --create-pass")
		}
		if *username == "" && *password == "" && mycnfUser == "" {
			log.Fatal("Admin credentials required. Use -u and -p flags or .my.cnf for admin connection")
		}
		log.Printf("%s Google Cloud SQL mode: will revoke cloudsqlsuperuser role after granting roles", green("[+]"))
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

	// Check if .my.cnf contains admin credentials when creating users
	if *createUser != "" {
		auth.CheckMyCnfCredentialsForAdmin()
	}
}

func checkConnection(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// connect to the database using the new package structure
func connectToDatabase() *database.Manager {
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
		mycnfHost, mycnfUser, mycnfPwd := auth.ReadMyCnf()
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

				// Check if host from config already contains parameters
				if strings.Contains(host, "?") {
					parts := strings.SplitN(host, "?", 2)
					hostname := parts[0]
					params := parts[1]

					// Add port to hostname part if not already present
					if !strings.Contains(hostname, ":") {
						if cfg.MySQL.Port != "" {
							hostname = fmt.Sprintf("%s:%s", hostname, cfg.MySQL.Port)
						} else {
							hostname = fmt.Sprintf("%s:3306", hostname)
						}
					}
					// Reassemble the host string
					host = fmt.Sprintf("%s?%s", hostname, params)
				} else {
					// No parameters, just check for port
					if !strings.Contains(host, ":") {
						if cfg.MySQL.Port != "" {
							host = fmt.Sprintf("%s:%s", host, cfg.MySQL.Port)
						} else {
							host = fmt.Sprintf("%s:3306", host)
						}
					}
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

	if *isGCP && strings.Contains(host, "cloud-sql") {
		log.Printf("%s GCP Cloud SQL detected - will handle cloudsqlsuperuser role", yellow("[!]"))
	}

	dsn = auth.BuildDSNWithParams(user, pwd, host)

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

	// Modified to pass host, username, and password to NewManager
	dbManager := database.NewManager(db, host, user, pwd)

	// Set authentication plugin if specified
	if *authPlugin != "" {
		if *authPlugin != "mysql_native_password" && *authPlugin != "caching_sha2_password" {
			log.Printf("%s Invalid authentication plugin: %s (must be mysql_native_password or caching_sha2_password)",
				yellow("[!]"), *authPlugin)
		} else {
			dbManager.AuthPlugin = *authPlugin
			log.Printf("%s Forcing authentication plugin: %s",
				yellow("[!]"), *authPlugin)
		}
	}

	// Disable password policy if flag is set
	if *skipPasswordPolicy {
		dbManager.Logger.Printf("%s Password policy enforcement disabled", yellow("[!]"))
		dbManager.PasswordPolicy.MinLength = 0
		dbManager.PasswordPolicy.RequireUppercase = false
		dbManager.PasswordPolicy.RequireLowercase = false
		dbManager.PasswordPolicy.RequireDigits = false
		dbManager.PasswordPolicy.RequireSpecialChars = false
	}

	return dbManager
}

// main function
func main() {
	// print help menu if no flags are provided or -h flag is set
	if len(os.Args) == 1 || *help {
		flag.Usage()
		os.Exit(0)
	}

	// Handle connection test first - this is separate from other operations
	if *testConnection {
		// Check required parameters
		if *testUser == "" {
			log.Fatalf("%s Missing required -user parameter for connection test", red("✘"))
		}
		if *testHost == "" {
			log.Fatalf("%s Missing required -host parameter for connection test", red("✘"))
		}

		// Call the test connection function
		log.Printf("%s Testing MySQL connection with provided credentials...", yellow("[!]"))
		err := database.TestConnection(*testHost, *testUser, *testPass)
		if err != nil {
			log.Fatalf("%s Connection test failed: %v", red("✘"), err)
		}
		log.Printf("%s Connection test successful!", green("[+]"))
		return
	}

	// connect to the source database
	dbManager := connectToDatabase()
	defer dbManager.DB.Close()

	// Early password validation with additional warning about special characters
	if *createUser != "" && *createPassword != "" && !*skipPasswordPolicy {
		// Create a validator using the same policy as the database manager
		policy := dbManager.PasswordPolicy

		// Set SQLFileMode flag if using SQL file execution
		if *useSQLFile {
			policy.SQLFileMode = true
			log.Printf("%s Pre-validating NEW USER password with relaxed policy (SQL file mode)...",
				yellow("[!]"))
		} else {
			log.Printf("%s Pre-validating NEW USER password against policy (min length: %d)...",
				yellow("[!]"), policy.MinLength)
		}

		// Check for problematic special characters
		if containsProblematicChars(*createPassword) {
			log.Printf("%s Warning: Password contains special characters that may need escaping when used with MySQL CLI",
				yellow("[!]"))
		}

		if *debugPassword {
			log.Printf("%s Debugging password characters:", yellow("[!]"))
			// Use the dedicated debug validation function instead
			if err := auth.ValidatePasswordWithDebug(*createPassword, policy); err != nil {
				log.Fatalf("%s Password policy violation for new user creation: %v", red("✘"), err)
			}
		} else {
			// Use the standard validation function for non-debug case
			if err := auth.ValidatePassword(*createPassword, policy); err != nil {
				log.Fatalf("%s Password policy violation for new user creation: %v", red("✘"), err)
			}
		}

		log.Printf("%s New user password pre-validation successful", green("[+]"))
	}

	// When using SQL file approach for complex passwords
	if *useSQLFile && *createUser != "" && *createPassword != "" {
		log.Printf("%s Using SQL file execution method for complex password handling", yellow("[!]"))

		// Extract host for SQL executor — strip DSN query params but keep port.
		hostForSQLFile := dbManager.Host
		if strings.Contains(hostForSQLFile, "?") {
			hostForSQLFile = strings.Split(hostForSQLFile, "?")[0]
		}

		// Create SQL file executor with proper credentials
		executor := database.NewSQLFileExecutor(
			hostForSQLFile,
			dbManager.Username,
			dbManager.Password,
			log.New(os.Stdout, "", log.LstdFlags))

		// Gather roles to grant - only create roles list if -r flag was provided
		var rolesToGrant []string
		if *role != "" {
			rolesToGrant = strings.Split(*role, ",")

			// Create roles first
			for _, r := range rolesToGrant {
				if err := dbManager.CreateRole(r); err != nil {
					log.Fatalf("Failed to create role: %v", err)
				}
			}
		}

		// Add debug logging for parameters being passed to ExecuteUserCreation
		log.Printf("%s Debug: Passing to SQL executor - dbName='%s', grants='%s', roles=%v",
			yellow("[!]"), *dbName, *grants, rolesToGrant)

		// Execute user creation via SQL file
		err := executor.ExecuteUserCreation(
			*createUser,
			*createPassword,
			*authPlugin,
			rolesToGrant,
			*dbName,
			*grants)

		if err != nil {
			log.Fatalf("%s Failed to execute SQL file: %v", red("✘"), err)
		}

		log.Printf("%s User creation via SQL file completed successfully", green("[+]"))

		// Commented out: Connection instructions block
		/*
			log.Printf("\n%s Connection instructions:", green("[+]"))
			log.Printf("To connect with MySQL client, use one of these methods:")
			log.Printf("1. mysql -h %s -u %s -p", host, *createUser)
			log.Printf("   (then enter password when prompted)")
			log.Printf("2. Create a ~/.my.cnf file with:")
			log.Printf("   [client]")
			log.Printf("   user=%s", *createUser)
			log.Printf("   password=%s", *createPassword)
			log.Printf("   host=%s", host)
		*/

		return
	}

	// Handle show commands first as they don't need transactions
	if *showRoleName != "" {
		// New flag for showing role grants directly
		if err := dbManager.ShowRoleGrants(*showRoleName); err != nil {
			log.Fatalf("Failed to show grants for role %s: %v", *showRoleName, err)
		}
		return
	}

	if *showGrants && *role != "" {
		// Keep existing behavior for -show -r role1,role2
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			if err := dbManager.ShowRoleGrants(r); err != nil {
				log.Fatalf("Failed to show grants: %v", err)
			}
		}
		return
	}

	if *showUserName != "" {
		if err := dbManager.ShowUserGrants(*showUserName); err != nil {
			log.Fatalf("Failed to show user grants: %v", err)
		}
		return
	}

	ctx := context.Background()
	// Start transaction
	if err := dbManager.BeginTx(ctx); err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	// When using GCP mode, ensure we create and configure roles first
	if *isGCP && *role != "" {
		// Create roles first
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			if err := dbManager.CreateRole(r); err != nil {
				log.Fatalf("Failed to create role: %v", err)
			}
		}

		// Grant privileges to roles if specified
		if *dbName != "" && *grants != "" {
			for _, r := range roles {
				if err := dbManager.GrantPrivileges(r, *dbName, *grants); err != nil {
					log.Fatalf("Failed to grant privileges: %v", err)
				}
			}
		}

		// Create user if specified
		if *createUser != "" && *createPassword != "" {
			_, err := dbManager.CreateUser(*createUser, *createPassword)
			if err != nil {
				log.Fatalf("Failed to create user: %v", err)
			}
		}

		// Grant roles to user
		targetUser := *createUser
		if targetUser == "" {
			targetUser = *username
		}
		for _, r := range roles {
			if err := dbManager.GrantRoles(targetUser, r, *isGCP); err != nil {
				log.Fatalf("Failed to grant role: %v", err)
			}
		}
	} else {
		// Non-GCP flow - existing logic
		// create roles
		if *role != "" {
			roles := strings.Split(*role, ",")
			for _, r := range roles {
				if err := dbManager.CreateRole(r); err != nil {
					log.Fatalf("Failed to create role: %v", err)
				}
			}
		}

		// grant privileges to roles
		if *role != "" && *dbName != "" && *grants != "" {
			roles := strings.Split(*role, ",")
			for _, r := range roles {
				if err := dbManager.GrantPrivileges(r, *dbName, *grants); err != nil {
					log.Fatalf("Failed to grant privileges: %v", err)
				}
			}
		}

		// create users
		if *createUser != "" && *createPassword != "" {
			users := strings.Split(*createUser, ",")
			passwords := strings.Split(*createPassword, ",")
			for i, u := range users {
				_, err := dbManager.CreateUser(u, passwords[i])
				if err != nil {
					log.Fatalf("Failed to create user: %v", err)
				}
			}
		}

		// grant roles to users
		if (*createUser != "" || *username != "") && *role != "" {
			var users []string
			if *createUser != "" {
				users = strings.Split(*createUser, ",")
			} else {
				users = strings.Split(*username, ",")
			}
			roles := strings.Split(*role, ",")
			for _, u := range users {
				for _, r := range roles {
					if err := dbManager.GrantRoles(u, r, false); err != nil {
						log.Fatalf("Failed to grant role: %v", err)
					}
				}
			}
		}

		// grant privileges to users
		if *createUser != "" && *dbName != "" && *grants != "" {
			users := strings.Split(*createUser, ",")
			for _, u := range users { // Fix: add "range" keyword here
				if err := dbManager.GrantPrivilegesToUser(u, *dbName, *grants); err != nil {
					log.Fatalf("Failed to grant privileges to user: %v", err)
				}
			}
		}

		// set default role for users
		if *createUser != "" && *role != "" {
			users := strings.Split(*createUser, ",")
			roles := strings.Split(*role, ",")
			for _, u := range users {
				for _, r := range roles {
					if err := dbManager.SetDefaultRole(u, r); err != nil {
						log.Fatalf("Failed to set default role: %v", err)
					}
				}
			}
		}
	}

	// Commit transaction
	if err := dbManager.CommitTx(); err != nil {
		if rbErr := dbManager.RollbackTx(); rbErr != nil {
			log.Printf("Failed to rollback: %v", rbErr)
		}
		log.Fatalf("Failed to commit transaction: %v", err)
	}
}

// Helper function to detect potentially problematic password characters
func containsProblematicChars(password string) bool {
	problematic := []string{"`", "$", "\\", "|", "&", ";", "<", ">", "(", ")", "*"}
	for _, char := range problematic {
		if strings.Contains(password, char) {
			return true
		}
	}
	return false
}
