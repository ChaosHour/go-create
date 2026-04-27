package auth

import (
	"strings"
	"testing"
)

func TestDefaultPasswordPolicy(t *testing.T) {
	policy := DefaultPasswordPolicy()

	if policy.MinLength != 30 {
		t.Errorf("DefaultPasswordPolicy().MinLength = %d, want 30", policy.MinLength)
	}
	if !policy.RequireUppercase {
		t.Error("DefaultPasswordPolicy().RequireUppercase = false, want true")
	}
	if !policy.RequireLowercase {
		t.Error("DefaultPasswordPolicy().RequireLowercase = false, want true")
	}
	if !policy.RequireDigits {
		t.Error("DefaultPasswordPolicy().RequireDigits = false, want true")
	}
	if !policy.RequireSpecialChars {
		t.Error("DefaultPasswordPolicy().RequireSpecialChars = false, want true")
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		policy   PasswordPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password meeting all requirements",
			password: "ValidP-ssw0rd123!WithSpecialChars",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: false,
		},
		{
			name:     "password too short",
			password: "Short1!",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: true,
			errMsg:  "at least 30 characters long",
		},
		{
			name:     "missing uppercase letter",
			password: "validp-ssw0rd123!withspecialchars",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: true,
			errMsg:  "uppercase letter",
		},
		{
			name:     "missing lowercase letter",
			password: "VALIDP-SSW0RD123!WITHSPECIALCHARS",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: true,
			errMsg:  "lowercase letter",
		},
		{
			name:     "missing digit",
			password: "ValidPassword!-WithSpecialCharsNo",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: true,
			errMsg:  "digit",
		},
		{
			name:     "missing special character",
			password: "ValidPassword123WithoutSpecialCharNow",
			policy: PasswordPolicy{
				MinLength:           35,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: true,
			errMsg:  "special character",
		},
		{
			name:     "relaxed policy - only length required",
			password: "shorterpasswordwithnospecialrules",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    false,
				RequireLowercase:    false,
				RequireDigits:       false,
				RequireSpecialChars: false,
			},
			wantErr: false,
		},
		{
			name:     "SQL file mode - allows single quotes",
			password: "ValidP-ssw0rd123!With'SingleQuotes",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
				SQLFileMode:         true,
			},
			wantErr: false,
		},
		{
			name:     "non-SQL file mode - rejects single quotes",
			password: "ValidP-ssw0rd123!With'SingleQuotes",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
				SQLFileMode:         false,
			},
			wantErr: true,
			errMsg:  "forbidden MySQL character",
		},
		{
			name:     "password with backslash - non-SQL mode",
			password: "ValidP-ssw0rd123!With\\BackslashNo",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
				SQLFileMode:         false,
			},
			wantErr: true,
			errMsg:  "forbidden MySQL character",
		},
		{
			name:     "password with double quotes - non-SQL mode",
			password: "ValidP-ssw0rd123!With\"DoubleQuotesNow",
			policy: PasswordPolicy{
				MinLength:           35,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
				SQLFileMode:         false,
			},
			wantErr: true,
			errMsg:  "forbidden MySQL character",
		},
		{
			name:     "exact minimum length",
			password: "ValidP-ssw0rd123!ExactlyLength",
			policy: PasswordPolicy{
				MinLength:           30,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: false,
		},
		{
			name:     "zero length requirement",
			password: "Short1!",
			policy: PasswordPolicy{
				MinLength:           0,
				RequireUppercase:    true,
				RequireLowercase:    true,
				RequireDigits:       true,
				RequireSpecialChars: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePassword() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestPasswordPolicyWithShellProblematicChars(t *testing.T) {
	// Test that shell-problematic characters generate warnings but don't fail validation
	policy := PasswordPolicy{
		MinLength:           30,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireDigits:       true,
		RequireSpecialChars: true,
		SQLFileMode:         true,
	}

	problematicPasswords := []string{
		"ValidP-ssw0rd123!With$DollarSign",
		"ValidP-ssw0rd123!With|PipeCharacter",
		"ValidP-ssw0rd123!With&Ampersand123",
		"ValidP-ssw0rd123!With<LessThanSign",
		"ValidP-ssw0rd123!With>GreaterThan1",
		"ValidP-ssw0rd123!With*AsteriskChar",
		"ValidP-ssw0rd123!With?QuestionMark",
		"ValidP-ssw0rd123!With!ExclamationM",
		"ValidP-ssw0rd123!With(Parentheses)",
		"ValidP-ssw0rd123!With`BacktickChar",
		"ValidP-ssw0rd123!With SpaceCharact",
	}

	for _, password := range problematicPasswords {
		t.Run("Shell problematic: "+password[:20]+"...", func(t *testing.T) {
			// Should not error out, just warn
			err := ValidatePassword(password, policy)
			if err != nil {
				t.Errorf("ValidatePassword() with shell-problematic char should warn but not error, got: %v", err)
			}
		})
	}
}

func TestPasswordPolicyForbiddenChars(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:           30,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireDigits:       true,
		RequireSpecialChars: true,
		SQLFileMode:         false, // Forbidden chars should cause errors
	}

	forbiddenPasswords := []struct {
		password string
		char     string
	}{
		{"ValidP-ssw0rd123!'SingleQuoteFail", "'"},
		{"ValidP-ssw0rd123!\"DoubleQuoteFail", "\""},
		{"ValidP-ssw0rd123!\\BackslashFailsH", "\\"},
		{"ValidP-ssw0rd123!;SemicolonFailsHe", ";"},
		{"ValidP-ssw0rd123!--CommentFailsHer", "--"},
		{"ValidP-ssw0rd123!#HashFailsHereNow", "#"},
		{"ValidP@ssw0rd123!AtSignFailsHereNow", "@"},
	}

	for _, tt := range forbiddenPasswords {
		t.Run("Forbidden char: "+tt.char, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if err == nil {
				t.Errorf("ValidatePassword() should fail with forbidden char '%s', but didn't", tt.char)
			}
			if !strings.Contains(err.Error(), "forbidden MySQL character") {
				t.Errorf("ValidatePassword() error should mention forbidden character, got: %v", err)
			}
		})
	}
}
