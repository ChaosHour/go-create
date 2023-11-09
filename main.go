package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	source   = flag.String("s", "", "Source Host")
	username = flag.String("u", "", "User")
	password = flag.String("p", "", "Password")
	host     = flag.String("host", "", "Host")
	grants   = flag.String("g", "", "Comma-separated list of grants to create")
	dbName   = flag.String("db", "", "Database name")
	//maxUserConnections = flag.Int("m", 0, "Max user connections")
	role = flag.String("r", "", "Comma-separated list of roles to create")
	help = flag.Bool("h", false, "Print help")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()

// var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()

//var blue = color.New(color.FgBlue).SprintFunc()

// parse flags
func init() {
	flag.Parse()
}

// global variables
var (
	db  *sql.DB
	err error
)

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

// connect to the source database and create a connection
func connectToDatabase() {

	db, err = sql.Open("mysql", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+*source+":3306)/")

	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Connecting to database:", os.Getenv("MYSQL_USER")+":"+os.Getenv("MYSQL_PASSWORD")+"@tcp("+*source+":3306)/mysql")
	//defer db1.Close()
}

// check if a MySQL user with the specified username already exists
func checkUserExists(db *sql.DB, username string) (bool, error) {
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM mysql.user WHERE User='%s'", username)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// create a function to check MySQL version
func checkMySQLVersion(db *sql.DB, minVersion string, roleName string) {
	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		log.Fatal(err)
	}

	if version < minVersion && roleName != "" {
		log.Fatalf("MySQL version is %s, but minimum required version is %s. Roles are not supported in this version.", version, minVersion)
	}
}

// create a function to check if a MySQL role already exists
func checkRoleExists(db *sql.DB, roleName string) (bool, error) {
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM mysql.roles_priv WHERE Role='%s'", roleName)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// create a function to create a role or roles
func createRole(db *sql.DB, role string) {
	// create the role
	_, err := db.Exec(fmt.Sprintf("CREATE ROLE %s", role))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Created role:", role)
}

func createUser(db *sql.DB, username string, host string, password string) {
	// Create a new user
	query := fmt.Sprintf("CREATE USER `%s`@`%s` IDENTIFIED BY '%s'", username, host, password)
	_, err := db.Exec(query)
	if err != nil {
		log.Println(err)
		//debug.PrintStack()
		os.Exit(1)
	}
	log.Println(green("[+]"), "User", username, "created successfully.")
}

// create a function to grant privileges to the role or roles
func grantPrivileges(db *sql.DB, role string, dbName string, grants string, host string) {
	//fmt.Println("grantPrivileges called") // add this line
	//fmt.Println("role:", role)            // add this line
	//fmt.Println("dbName:", dbName)        // add this line
	//fmt.Println("grants:", grants)        // add this line
	//fmt.Println("host:", host)            // add this line

	// grant privileges to the role
	var grantQuery string
	if dbName == "*.*" {
		grantQuery = fmt.Sprintf("GRANT %s ON *.* TO `%s`@`%s`", grants, role, host)
	} else {
		grantQuery = fmt.Sprintf("GRANT %s ON %s.* TO `%s`@`%s`", grants, dbName, role, host)
	}

	fmt.Println("grantQuery:", grantQuery) // add this line
	_, err := db.Exec(grantQuery)
	if err != nil {
		log.Println(err)
		//debug.PrintStack()
		os.Exit(1)
	}
	log.Println(green("[+]"), "Granted privileges to role:", role)
}

// create a function to grant roles to the user or users
func grantRoles(db *sql.DB, username string, role string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT '%s' TO '%s'", role, username))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted role to user:", username)
}

// create a function to grant privileges to the user or users
func grantPrivilegesToUser(db *sql.DB, username string, dbName string, grants string) {
	fmt.Println("role:", *role)
	if username == *role {
		return
	}
	// Check code here for dbName and *.* Kurt Larsen
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT %s ON '%s' TO '%s'", grants, dbName, username))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted privileges to user:", username)
}

// function to add SET DEFAULT ROLE to the user
func setDefaultRole(db *sql.DB, username string, role string) {
	_, err := db.Exec(fmt.Sprintf("ALTER USER '%s' DEFAULT ROLE '%s'", username, role))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Set default role for user:", username)
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
	connectToDatabase()

	// check MySQL version
	checkMySQLVersion(db, "8.0.0", *role)

	// Create user or role based on command line arguments
	if *username != "" {
		// check if the user already exists. If the -u flag is set check if the user exists.
		userExists, err := checkUserExists(db, *username)
		if err != nil {
			log.Fatal(err)
		}
		if userExists {
			// if the user exists, update grants
			log.Println(yellow("[!]"), "User", *username, "already exists. Updating grants...")
		} else {
			// if the user does not exist, create the user
			createUser(db, *username, *host, *password)
		}
		// grant privileges to the user
		grantPrivileges(db, *username, *dbName, *grants, *host)
	}

	// create role or roles
	if *role != "" {
		roleExists, err := checkRoleExists(db, *role)
		if err != nil {
			log.Fatal(err)
		}
		if !roleExists {
			roles := strings.Split(*role, ",")
			for _, r := range roles {
				createRole(db, r)
			}
		}
		// grant privileges to the role
		grantPrivileges(db, *role, *dbName, *grants, *host)
	}

	// grant roles to users
	if *username != "" && *role != "" {
		users := strings.Split(*username, ",")
		roles := strings.Split(*role, ",")
		for _, u := range users {
			for _, r := range roles {
				grantRoles(db, u, r)
			}
		}
	}

	// grant privileges to users
	if *username != "" && *dbName != "" && *grants != "" && *role != "" {
		users := strings.Split(*username, ",")
		for _, u := range users {
			grantPrivilegesToUser(db, u, *dbName, *grants)
		}
	}

	// set default role for users
	if *username != "" && *role != "" {
		users := strings.Split(*username, ",")
		roles := strings.Split(*role, ",")
		for _, u := range users {
			setDefaultRole(db, u, roles[0])
		}
	}
	// close the database connection
	defer db.Close()
}
