package database

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

const querySelectVersion = "SELECT VERSION()"

func TestGetMySQLVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	// Update the constructor call with mock parameters
	manager := NewManager(db, "localhost:3306", "testuser", "testpassword")

	// Test MySQL 5.7
	mock.ExpectQuery(querySelectVersion).WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow("5.7.35"),
	)

	version, err := manager.GetMySQLVersion()
	if err != nil {
		t.Fatalf("Error getting version: %v", err)
	}
	if version != 57 {
		t.Errorf("Expected version 57, got %d", version)
	}

	// Test MySQL 8.0
	mock.ExpectQuery(querySelectVersion).WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow("8.0.28"),
	)

	version, err = manager.GetMySQLVersion()
	if err != nil {
		t.Fatalf("Error getting version: %v", err)
	}
	if version != 80 {
		t.Errorf("Expected version 80, got %d", version)
	}
}

func TestCheckUserExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	// Update the constructor call with mock parameters
	manager := NewManager(db, "localhost:3306", "testuser", "testpassword")

	// Test user exists
	mock.ExpectQuery("SELECT COUNT.*FROM mysql.user").
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery("SELECT Host FROM mysql.user").
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"host"}).AddRow("localhost"))

	exists, host, err := manager.CheckUserExists("testuser")
	if err != nil {
		t.Fatalf("Error checking user: %v", err)
	}
	if !exists {
		t.Errorf("Expected user to exist")
	}
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", host)
	}
}

func TestShowRoleGrants(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	// Update the constructor call with mock parameters
	manager := NewManager(db, "localhost:3306", "testuser", "testpassword")

	mock.ExpectQuery("SHOW GRANTS FOR").
		WillReturnRows(sqlmock.NewRows([]string{"grants"}).
			AddRow("GRANT SELECT ON db.* TO 'role'").
			AddRow("GRANT INSERT ON db.* TO 'role'"))

	err = manager.ShowRoleGrants("role")
	if err != nil {
		t.Fatalf("Error showing grants: %v", err)
	}
}

func TestBeginTxCommitRollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock: %v", err)
	}
	defer db.Close()

	// Update the constructor call with mock parameters
	manager := NewManager(db, "localhost:3306", "testuser", "testpassword")
	ctx := context.Background()

	mock.ExpectBegin()
	err = manager.BeginTx(ctx)
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	mock.ExpectCommit()
	err = manager.CommitTx()
	if err != nil {
		t.Fatalf("Error committing transaction: %v", err)
	}

	// Test rollback in a separate transaction
	mock.ExpectBegin()
	err = manager.BeginTx(ctx)
	if err != nil {
		t.Fatalf("Error beginning transaction: %v", err)
	}

	mock.ExpectRollback()
	err = manager.RollbackTx()
	if err != nil {
		t.Fatalf("Error rolling back transaction: %v", err)
	}
}
