package auth

import (
	"fmt"
	"strings"
	"unicode"
)

// PasswordPolicy defines requirements for password strength
// NOTE: This policy only applies when creating new users with -create-user and -create-pass
// It does NOT affect authentication credentials from .my.cnf or command line flags
type PasswordPolicy struct {
	MinLength           int
	RequireUppercase    bool
	RequireLowercase    bool
	RequireDigits       bool
	RequireSpecialChars bool
	SQLFileMode         bool
}

// ForbiddenPasswordChars defines characters that should normally be avoided in MySQL passwords
// but will be allowed when using the -use-sql-file flag
var ForbiddenPasswordChars = []string{"'", "\"", "\\", ";", "--", "#", "@"}

// ShellProblematicChars defines characters that cause issues in command-line MySQL operations
var ShellProblematicChars = []string{"$", "|", "&", "<", ">", "*", "?", "!", "(", ")", "`", " "}

// DefaultPasswordPolicy returns the application's default password policy
// for new user creation only
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:           30,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireDigits:       true,
		RequireSpecialChars: true,
	}
}

// ValidatePassword checks if a password meets the specified policy
// This is only used for new user creation, not for authentication
func ValidatePassword(password string, policy PasswordPolicy) error {
	// Add debug output
	fmt.Printf("DEBUG: Validating password length: %d against policy min: %d\n",
		len(password), policy.MinLength)

	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long (got %d)",
			policy.MinLength, len(password))
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case !unicode.IsLetter(char) && !unicode.IsDigit(char) && !unicode.IsSpace(char):
			hasSpecial = true
		}
	}

	if policy.RequireUppercase && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if policy.RequireLowercase && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if policy.RequireDigits && !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if policy.RequireSpecialChars && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Return warnings but don't fail for special characters when SQLFileMode is true
	if !policy.SQLFileMode {
		// Check for forbidden MySQL password characters - only fail if not using SQL file mode
		for _, char := range ForbiddenPasswordChars {
			if strings.Contains(password, char) {
				return fmt.Errorf("password contains forbidden MySQL character: '%s' - use -use-sql-file flag or see docs/mysql_password_guidelines.md", char)
			}
		}
	}

	// Check for shell-problematic characters (warning only)
	for _, char := range ShellProblematicChars {
		if strings.Contains(password, char) {
			fmt.Printf("WARNING: Password contains shell-problematic character: '%s' which may cause connection issues\n", char)
			if !policy.SQLFileMode {
				fmt.Println("Consider using -use-sql-file flag and see docs/mysql_password_guidelines.md")
			}
			break // Only warn once
		}
	}

	return nil
}
