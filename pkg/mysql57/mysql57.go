package mysql57

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	//source   = flag.String("s", "", "Source Host")
	username = flag.String("u", "", "User")
	password = flag.String("p", "", "Password")
	host     = flag.String("host", "", "Host to assign to the user (default: %)")
	grants   = flag.String("g", "", "Comma-separated list of grants to create")
	dbName   = flag.String("db", "", "Database name")
	//role     = flag.String("r", "", "Comma-separated list of roles to create")
	//help     = flag.Bool("h", false, "Print help")
)

// define colors
var green = color.New(color.FgGreen).SprintFunc()

// var red = color.New(color.FgRed).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()

//var blue = color.New(color.FgBlue).SprintFunc()

// global variables
var (
	db  *sql.DB
	err error
)

// create a function to create a role or roles
func createRole(db *sql.DB, role string) {
	// check if the role already exists
	exists, err := CheckUserExists(db, role)
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
func GrantPrivileges(db *sql.DB, role string, dbName string, grants string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, role))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted privileges to role:", role)
}

// create a function to create a user or users
func CreateUser(db *sql.DB, username string, password string) {
	// check if the user already exists
	exists, err := CheckUserExists(db, username)
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
func GrantRoles(db *sql.DB, username string, role string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT `%s` TO `%s`", role, username))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted role to user:", username)
}

/*
// create a function to grant privileges to the user or users
func grantPrivilegesToUser(db *sql.DB, username string, dbName string, grants string) {
	// grant privileges to the role
	_, err := db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, username))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Granted privileges to user:", username)
}

*/
// create a function to grant privileges to the user or users
func GrantPrivilegesToUser(db *sql.DB, username string, dbName string, grants string) {
	// grant privileges to the user
	if dbName == "*.*" {
		_, err := db.Exec(fmt.Sprintf("GRANT %s ON *.* TO `%s`", grants, username))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		_, err := db.Exec(fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`", grants, dbName, username))
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println(green("[+]"), "Granted privileges to user:", username)
}

// function to add SET DEFAULT ROLE to the user
func SetDefaultRole(db *sql.DB, username string, role string) {
	_, err := db.Exec(fmt.Sprintf("ALTER USER `%s` DEFAULT ROLE `%s`", username, role))
	if err != nil {
		log.Fatal(err)
	}
	log.Println(green("[+]"), "Set default role for user:", username)
}
