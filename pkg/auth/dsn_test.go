package auth

import (
	"errors"
	"testing"
)

func TestBuildDSNWithParams(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		hostString string
		want       string
	}{
		{
			name:       "basic host with default port",
			username:   "testuser",
			password:   "testpass",
			hostString: "localhost",
			want:       "testuser:testpass@tcp(localhost:3306)/",
		},
		{
			name:       "host with custom port",
			username:   "root",
			password:   "secret",
			hostString: "192.168.1.100:3307",
			want:       "root:secret@tcp(192.168.1.100:3307)/",
		},
		{
			name:       "host with URL parameters",
			username:   "admin",
			password:   "pass123",
			hostString: "db.example.com?charset=utf8mb4&parseTime=true",
			want:       "admin:pass123@tcp(db.example.com:3306)/?charset=utf8mb4&parseTime=true",
		},
		{
			name:       "host with port and URL parameters",
			username:   "user",
			password:   "pwd",
			hostString: "mysql.host.com:3308?timeout=5s",
			want:       "user:pwd@tcp(mysql.host.com:3308)/?timeout=5s",
		},
		{
			name:       "complex password with special characters",
			username:   "dbuser",
			password:   "p@ssw0rd!#$",
			hostString: "localhost",
			want:       "dbuser:p@ssw0rd!#$@tcp(localhost:3306)/",
		},
		{
			name:       "empty password",
			username:   "nopass",
			password:   "",
			hostString: "localhost:3306",
			want:       "nopass:@tcp(localhost:3306)/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDSNWithParams(tt.username, tt.password, tt.hostString)
			if got != tt.want {
				t.Errorf("BuildDSNWithParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "basic DSN with password",
			dsn:  "user:secretpass@tcp(localhost:3306)/",
			want: "user:****@tcp(localhost:3306)/",
		},
		{
			name: "DSN with complex password",
			dsn:  "admin:p@ssw0rd!123@tcp(db.example.com:3307)/mydb",
			want: "admin:****@tcp(db.example.com:3307)/mydb",
		},
		{
			name: "DSN with parameters",
			dsn:  "root:mypass@tcp(localhost:3306)/?charset=utf8",
			want: "root:****@tcp(localhost:3306)/?charset=utf8",
		},
		{
			name: "DSN without password (malformed)",
			dsn:  "user@tcp(localhost:3306)/",
			want: "user@tcp(localhost:3306)/",
		},
		{
			name: "empty DSN",
			dsn:  "",
			want: "",
		},
		{
			name: "DSN with empty password",
			dsn:  "user:@tcp(localhost:3306)/",
			want: "user:****@tcp(localhost:3306)/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeDSN(tt.dsn)
			if got != tt.want {
				t.Errorf("SanitizeDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		password string
		want     string
	}{
		{
			name:     "error with password in message",
			err:      errors.New("connection failed: password 'secret123' is invalid"),
			password: "secret123",
			want:     "connection failed: password '****' is invalid",
		},
		{
			name:     "error with password multiple times",
			err:      errors.New("tried secret123 but secret123 failed"),
			password: "secret123",
			want:     "tried **** but **** failed",
		},
		{
			name:     "error without password",
			err:      errors.New("connection timeout"),
			password: "secret123",
			want:     "connection timeout",
		},
		{
			name:     "nil error",
			err:      nil,
			password: "anypass",
			want:     "",
		},
		{
			name:     "empty password",
			err:      errors.New("some error with text"),
			password: "",
			want:     "some error with text",
		},
		{
			name:     "complex password with special chars",
			err:      errors.New("auth failed with p@ss!123"),
			password: "p@ss!123",
			want:     "auth failed with ****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeError(tt.err, tt.password)
			if got == nil {
				if tt.want != "" {
					t.Errorf("SanitizeError() = nil, want error with message %v", tt.want)
				}
			} else if got.Error() != tt.want {
				t.Errorf("SanitizeError() = %v, want %v", got.Error(), tt.want)
			}
		})
	}
}

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		password string
		host     string
		want     string
	}{
		{
			name:     "basic connection",
			user:     "root",
			password: "pass",
			host:     "localhost:3306",
			want:     "root:pass@tcp(localhost:3306)/",
		},
		{
			name:     "with special characters",
			user:     "admin",
			password: "p@ss#123",
			host:     "db.local:3307",
			want:     "admin:p@ss#123@tcp(db.local:3307)/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDSN(tt.user, tt.password, tt.host)
			if got != tt.want {
				t.Errorf("BuildDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}
