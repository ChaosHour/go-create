package auth

import (
	"fmt"
	"strings"
)

// DumpPasswordCharacters creates a debug representation of a password
// showing each character and its ASCII/Unicode code
func DumpPasswordCharacters(password string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Password length: %d\n", len(password)))
	builder.WriteString("Character breakdown:\n")

	for i, char := range password {
		builder.WriteString(fmt.Sprintf("  Pos %2d: '%c' (Unicode: U+%04X, Decimal: %d)\n",
			i, char, char, char))
	}

	builder.WriteString("\nTo avoid shell interpretation of special characters, you can:\n")
	builder.WriteString("1. Use single quotes: 'complex!p@ssw0rd'\n")
	builder.WriteString("2. Escape special characters with backslash: complex\\!p@ssw0rd\n")
	builder.WriteString("3. Store your password in a file and use: -create-pass \"$(cat password.txt)\"\n")

	return builder.String()
}

// ValidatePasswordWithDebug enhances password validation with character-level debugging
// This function calls the standard ValidatePassword function but adds detailed diagnostics
func ValidatePasswordWithDebug(password string, policy PasswordPolicy) error {
	// Print the debug info about the password characters
	fmt.Println(DumpPasswordCharacters(password))

	// Call the standard validation function
	return ValidatePassword(password, policy)
}
