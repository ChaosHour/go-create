// Create functions for GRANTS for version MySQL 5.6

package mysql56

import (
	"database/sql"
	"flag"
	"fmt"

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

// GRANTS for MySQL 5.6 No roles just GRANTS assigned using GRANT statements
// DB may be *.* or dbname.*
// GRANTS may be ALL PRIVILEGES or a comma-separated list of privileges such as SELECT,INSERT,UPDATE,DELETE
// username is the username to create
// password is the password to assign to the user
// host is the host to assign to the user
// Example of creating a user with the GRANT statement an IDENTIFIED BY clause using the hashed password.
// GRANT PROCESS, RELOAD, REPLICATION CLIENT, SELECT, SUPER ON *.* TO 'pmm'@'127.0.0.1' IDENTIFIED BY PASSWORD '*CC44899BBE450A06A0823407493390266377825C';
// GRANT PROCESS, RELOAD, REPLICATION CLIENT, SELECT, SUPER ON *.* TO 'pmm'@'%' IDENTIFIED BY PASSWORD '*CC44899BBE450A06A0823407493390266377825C';
// Example of creating a user with the GRANT statement using a plain text password.
// GRANT SELECT, INSERT, UPDATE, DELETE ON smite_rivals.* TO 'smite'@'localhost' IDENTIFIED BY 'reACT4!4!';
// createUserAndGrantPrivileges creates a user and grants them privileges
func CreateUserAndGrantPrivileges(db *sql.DB, hashed bool) error {
	var query string

	// Check if password is hashed
	if hashed {
		query = fmt.Sprintf("GRANT %s ON %s TO '%s'@'%s' IDENTIFIED BY PASSWORD '%s'", *grants, *dbName, *username, *host, *password)
	} else {
		query = fmt.Sprintf("GRANT %s ON %s TO '%s'@'%s' IDENTIFIED BY '%s'", *grants, *dbName, *username, *host, *password)
	}

	// Execute the query
	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
