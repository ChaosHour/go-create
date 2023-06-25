package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
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
	grants   = flag.String("g", "", "Comma-separated list of grants to create")
	dbName   = flag.String("db", "", "Database name")
	role     = flag.String("r", "", "Comma-separated list of roles to create")
	help     = flag.Bool("h", false, "Print help")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()

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
	file, err := ioutil.ReadFile(os.Getenv("HOME") + "/.my.cnf")
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

// create a function to create a role or roles
func createRole(db *sql.DB, role string) {
	// check if the role already exists
	exists, err := checkUserExists(db, role)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		log.Println(yellow("[!]"), "Role", role, "already exists")
	} else {
		// create the role
		_, err := db.Exec(fmt.Sprintf("CREATE ROLE `%s`", role))
		if err != nil {
			log.Fatal(err)
		}
		log.Println(green("[+]"), "Created role:", role)
	}
}

// create a function to grant privileges to the role or roles
func grantPrivileges(db *sql.DB, role string, dbName string, grants string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted privileges to role:", role)
}

// create a function to create a user or users
func createUser(db *sql.DB, username string, password string) {
	// check if the user already exists
	exists, err := checkUserExists(db, username)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		log.Println(yellow("[!]"), "User", username, "already exists")
	} else {
		// create the user
		_, err := db.Exec(fmt.Sprintf("CREATE USER `%s` IDENTIFIED BY '%s'", username, password))
		if err != nil {
			log.Fatal(err)
		}
		log.Println(green("[+]"), "Created user:", username)
	}
}

// create a function to grant roles to the user or users
func grantRoles(db *sql.DB, username string, role string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT `%s` TO `%s`", role, username))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted role to user:", username)
}

// create a function to grant privileges to the user or users
func grantPrivilegesToUser(db *sql.DB, username string, dbName string, grants string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, username))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted privileges to user:", username)
}

// function to add SET DEFAULT ROLE to the user
func setDefaultRole(db *sql.DB, username string, role string) {
	_, err := db.Exec(fmt.Sprintf("ALTER USER `%s` DEFAULT ROLE `%s`", username, role))
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

	// create roles
	if *role != "" {
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			createRole(db, r)
		}
	}

	// grant privileges to roles
	if *role != "" && *dbName != "" && *grants != "" {
		roles := strings.Split(*role, ",")
		for _, r := range roles {
			grantPrivileges(db, r, *dbName, *grants)
		}
	}

	// create users
	if *username != "" && *password != "" {
		users := strings.Split(*username, ",")
		passwords := strings.Split(*password, ",")
		for i, u := range users {
			createUser(db, u, passwords[i])
		}
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
	if *username != "" && *dbName != "" && *grants != "" {
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
			for _, r := range roles {
				setDefaultRole(db, u, r)
			}
		}
	}

	// close the database connection
	defer db.Close()
}
