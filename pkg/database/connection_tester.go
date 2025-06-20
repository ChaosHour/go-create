package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ChaosHour/go-create/pkg/auth"
)

// TestConnection attempts to connect to a MySQL server using the provided credentials
func TestConnection(host, username, password string) error {
	log.Printf("Testing connection to %s as user %s...", host, username)

	// Create DSN for test connection using the centralized BuildDSN function
	dsn := auth.BuildDSNWithParams(username, password, host)

	// Add a timeout parameter to the DSN
	if strings.Contains(dsn, "?") {
		dsn += "&timeout=5s"
	} else {
		dsn += "?timeout=5s"
	}

	// Open a test connection
	testDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize connection: %w", err)
	}
	defer testDB.Close()

	// Set a short timeout
	testDB.SetConnMaxLifetime(5 * time.Second)

	// Try to ping the database
	if err := testDB.Ping(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// If successful, try a simple query
	var version string
	err = testDB.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return fmt.Errorf("connection succeeded but query failed: %w", err)
	}

	log.Printf("Connection successful! MySQL version: %s", version)
	return nil
}
