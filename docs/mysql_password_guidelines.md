# MySQL Password Guidelines

This document explains password requirements and common issues when using `go-create` to create MySQL users.

## Password Policy for New Users

When creating a new user with `--create-user` and `--create-pass`, the password **must** meet all of these requirements:

| Requirement        | Detail                                  |
| ------------------ | --------------------------------------- |
| Minimum length     | 30 characters                           |
| Uppercase letters  | At least one (A–Z)                      |
| Lowercase letters  | At least one (a–z)                      |
| Digits             | At least one (0–9)                      |
| Special characters | At least one non-alphanumeric character |

> **Important:** This policy only applies to new user passwords. It does **not** apply to admin connection credentials (`-u`/`-p` flags, `.my.cnf`, or config file passwords).

### Valid Password Examples

```text
MyStr0ngP@ssw0rd2024!ForLisa#       ✅ 30+ chars, all requirements met
Repl!c@t10nP@ssw0rd2024$ecure       ✅ 30+ chars, all requirements met
App_Wr1te#P@ssword2024!Secure%Key   ✅ 30+ chars, all requirements met
```

### Invalid Password Examples

```text
MyP@ss1!                             ❌ Too short (8 chars)
mysecurepasswordthatislong2024!      ❌ No uppercase letter
MYSECUREPASSWORDTHATISLONG2024!      ❌ No lowercase letter
MySecurePasswordThatIsLong!!         ❌ No digit
MySecurePassword123456789012345      ❌ No special character
```

### Bypassing the Policy

Use `-skip-password-policy` for testing or legacy password scenarios:

```bash
go-create --create-user testuser --create-pass "short" -skip-password-policy
```

---

## Forbidden Characters (Standard Mode)

Without the `-use-sql-file` flag, the following characters in new user passwords will cause an error because they conflict with MySQL's SQL syntax:

| Character | Why it's forbidden              |
| --------- | ------------------------------- |
| `'`       | String delimiter in SQL         |
| `"`       | String delimiter in SQL         |
| `\`       | SQL escape character            |
| `;`       | Statement terminator            |
| `--`      | SQL comment                     |
| `#`       | SQL comment                     |
| `@`       | Conflicts with user@host syntax |

### Solution: Use `-use-sql-file` for Complex Passwords

The `-use-sql-file` flag bypasses all shell and SQL-delimiter issues by using the Go MySQL driver directly with safe password encoding:

```bash
go-create --create-user myuser \
  --create-pass "Complex'P@ss\"w0rd!2024#With@AllChars;" \
  -use-sql-file -r app_role -db mydb -g select
```

---

## Shell-Problematic Characters

The following characters may behave unexpectedly when passed via the command line because the shell interprets them before they reach the program:

| Character | Shell meaning        |
| --------- | -------------------- |
| `$`       | Variable expansion   |
| `\|`      | Pipe                 |
| `&`       | Background job       |
| `<` `>`   | Redirection          |
| `*`       | Glob                 |
| `?`       | Glob                 |
| `(`  `)`  | Subshell             |
| `` ` ``   | Command substitution |
| (space)   | Argument separator   |

### Solutions

**Option 1: Use single quotes** to prevent shell expansion:

```bash
go-create --create-user myuser --create-pass 'My$Passw0rd!2024WithDollars'
```

**Option 2: Use `-use-sql-file`** (most reliable for complex passwords):

```bash
go-create --create-user myuser --create-pass 'My$Passw0rd!2024WithDollars' -use-sql-file
```

**Option 3: Store the password in a config file** (no shell quoting issues):

```json
{
  "mysql": {
    "host": "localhost",
    "user": "admin",
    "password": "adminpass"
  }
}
```

Then pass the new user's password via `-use-sql-file` to avoid shell interpretation.

---

## Diagnosing Password Issues

### Check which characters are causing problems

Use the `-debug-password` flag to get a character-by-character breakdown:

```bash
go-create --create-user myuser --create-pass 'MyP@ssw0rd!' \
  -debug-password -skip-password-policy
```

### Error: `password must be at least 30 characters long`

Your password is too short. Count the characters and pad to at least 30.

### Error: `password contains forbidden MySQL character: '@'`

Use `-use-sql-file`:

```bash
go-create --create-user myuser --create-pass 'User@Domain.com!Pass2024#Secure' \
  -use-sql-file -r myrole -db mydb -g select
```

### Warning: `Password contains shell-problematic character`

Wrap the password in single quotes or use `-use-sql-file`.

---

## Authentication Plugins

MySQL 5.7 uses `mysql_native_password` by default. MySQL 8.0+ uses `caching_sha2_password`.

`go-create` detects the server version and selects the correct plugin automatically. You can override this with the `-auth-plugin` flag:

```bash
# Force legacy plugin (useful for older clients)
go-create --create-user myuser --create-pass 'MyStr0ng!Pass2024#Secure' \
  -auth-plugin mysql_native_password -r myrole -db mydb -g select
```

Valid values: `mysql_native_password`, `caching_sha2_password`
