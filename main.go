package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github/ChaosHour/go-create/pkg/mysql56"
	_ "github/ChaosHour/go-create/pkg/mysql57"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
)

// Define flags
var (
	source   = flag.String("s", "", "Source Host")
	username = flag.String("u", "", "User")
	password = flag.String("p", "", "Password")
	host     = flag.String("host", "", "Host to assign to the user (default: %)")
	grants   = flag.String("g", "", "Comma-separated list of grants to create")
	dbName   = flag.String("db", "", "Database name")
	role     = flag.String("r", "", "Comma-separated list of roles to create")
	help     = flag.Bool("h", false, "Print help")
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

// Check what version of MySQL is running on the source host
func checkMySQLVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return "", err
	}
	return version, nil
}

// Check if the user has the required privileges to create users and grant privileges
func checkPrivileges(db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow("SELECT count(*) FROM mysql.user WHERE User='root' AND (GRANT_PRIV='Y' AND SUPER_PRIV='Y')").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// main function
func main() {
	// Print help if no arguments are supplied
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	// Print help if -h flag is supplied
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Connect to the database
	connectToDatabase()

	// Depending on the version of MySQL running on the source host, import the correct package
	version, err := checkMySQLVersion(db)
	if err != nil {
		log.Fatal(err)
	}
	if strings.HasPrefix(version, "5.6") {
		log.Println(green("[+]"), "MySQL version:", version)
		log.Println(green("[+]"), "Using MySQL 5.6")
		// import56()
		// err := CreateUserAndGrantPrivileges(db, true)
		// if err != nil {
		// 	log.Fatal(err)
		// }
	}
	if strings.HasPrefix(version, "5.7") {
		log.Println(green("[+]"), "MySQL version:", version)
		log.Println(green("[+]"), "Using MySQL 5.7")
		// import57()
	}
}
