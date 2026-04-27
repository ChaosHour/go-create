# Troubleshooting Guide

This guide helps resolve common issues when using go-create.

## Table of Contents

- [Connection Issues](#connection-issues)
- [Password Issues](#password-issues)
- [Permission Issues](#permission-issues)
- [Role and Grant Issues](#role-and-grant-issues)
- [Google Cloud SQL Issues](#google-cloud-sql-issues)
- [General Issues](#general-issues)

## Connection Issues

### Cannot Connect to MySQL Server

**Error:** `Failed to connect: dial tcp: connect: connection refused`

**Solutions:**
1. Verify MySQL server is running:
   ```bash
   mysql -h hostname -u root -p -e "SELECT 1"
   ```

2. Check host and port are correct:
   ```bash
   go-create -s hostname:3306 -u root -p password -show-user root
   ```

3. Verify firewall allows connections on port 3306

4. Check MySQL bind-address in my.cnf:
   ```
   bind-address = 0.0.0.0  # Allow remote connections
   ```

### Connection Timeout

**Error:** `Failed to connect: context deadline exceeded`

**Solutions:**
1. Add timeout parameter to connection string:
   ```json
   {
     "mysql": {
       "host": "hostname:3306?timeout=30s",
       "user": "root",
       "password": "password"
     }
   }
   ```

2. Check network connectivity:
   ```bash
   ping hostname
   telnet hostname 3306
   ```

### Authentication Failed

**Error:** `Access denied for user 'root'@'hostname'`

**Solutions:**
1. Verify credentials are correct
2. Check user exists and has proper host:
   ```sql
   SELECT user, host FROM mysql.user WHERE user='root';
   ```

3. Ensure admin user has sufficient privileges:
   ```sql
   GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION;
   FLUSH PRIVILEGES;
   ```

## Password Issues

### Password Policy Violation

**Error:** `password must be at least 30 characters long`

**Solutions:**
1. Use a longer password meeting requirements:
   - Minimum 30 characters
   - At least one uppercase letter
   - At least one lowercase letter
   - At least one digit
   - At least one special character

2. Skip policy for testing (not recommended for production):
   ```bash
   go-create --create-user test --create-pass "short" -skip-password-policy
   ```

3. Example of valid password:
   ```bash
   go-create --create-user myuser \
     --create-pass "MySecureP@ssw0rd2024WithManyChars!" \
     -r app_role -db mydb -g select
   ```

### Forbidden Character in Password

**Error:** `password contains forbidden MySQL character: '@'`

**Solutions:**
1. Use the SQL file mode for complex passwords:
   ```bash
   go-create --create-user myuser \
     --create-pass "Complex'P@ss\"w0rd!" \
     -use-sql-file -r app_role -db mydb -g select
   ```

2. Choose a password without forbidden characters:
   - Avoid: `'` `"` `\` `;` `--` `#` `@`
   - Safe special chars: `!` `%` `^` `&` `*` `-` `_` `+` `=`

3. Example safe password:
   ```bash
   go-create --create-user myuser \
     --create-pass "MySecure!P%ssw0rd2024^WithChars*" \
     -r app_role
   ```

### Password with Shell Special Characters

**Warning:** `Password contains shell-problematic character`

**Solutions:**
1. Use SQL file mode (recommended):
   ```bash
   go-create --create-user myuser --create-pass 'Pass$123' -use-sql-file
   ```

2. Use single quotes to prevent shell expansion:
   ```bash
   go-create --create-user myuser --create-pass 'Pass$123'
   ```

3. Use a config file instead:
   ```json
   {
     "mysql": {
       "host": "localhost",
       "user": "admin",
       "password": "admin_pass"
     }
   }
   ```

## Permission Issues

### Insufficient Privileges

**Error:** `Access denied; you need (at least one of) the CREATE USER privilege(s)`

**Solutions:**
1. Ensure admin user has CREATE USER privilege:
   ```sql
   GRANT CREATE USER ON *.* TO 'admin'@'%';
   FLUSH PRIVILEGES;
   ```

2. Check current privileges:
   ```bash
   go-create -show-user admin
   ```

3. Use root or admin account with full privileges

### Cannot Grant Privileges

**Error:** `Access denied for user 'admin'@'%' with GRANT OPTION`

**Solutions:**
1. Admin user needs GRANT OPTION:
   ```sql
   GRANT ALL PRIVILEGES ON *.* TO 'admin'@'%' WITH GRANT OPTION;
   FLUSH PRIVILEGES;
   ```

2. Verify GRANT OPTION is present:
   ```sql
   SHOW GRANTS FOR 'admin'@'%';
   ```

## Role and Grant Issues

### Roles Not Supported

**Warning:** `Roles are not supported in MySQL 5.7, skipping role creation`

**Explanation:**
MySQL 5.7 does not support roles. Roles were introduced in MySQL 8.0.

**Solutions:**
1. Upgrade to MySQL 8.0+ to use roles

2. For MySQL 5.7, use direct grants instead:
   ```bash
   # MySQL 5.7 - grant directly to user
   go-create --create-user myuser --create-pass "password..." \
     -g select,insert,update,delete -db myapp
   ```

3. For MySQL 8.0+, use roles:
   ```bash
   # MySQL 8.0+ - use roles
   go-create --create-user myuser --create-pass "password..." \
     -r app_role -g select,insert,update,delete -db myapp
   ```

### Role Already Exists

**Warning:** `Role app_read already exists`

**Explanation:**
The role you're trying to create already exists. This is not an error.

**Solutions:**
1. Show existing role grants:
   ```bash
   go-create -r app_read -show
   ```

2. Grant existing role to user:
   ```bash
   go-create --create-user newuser --create-pass "password..." -r app_read
   ```

3. Drop and recreate if needed:
   ```sql
   DROP ROLE IF EXISTS 'app_read';
   ```

### User Already Has Role

**Info:** Tool will not duplicate role grants if user already has the role.

**To verify:**
```bash
go-create -show-user username
```

## Google Cloud SQL Issues

### cloudsqlsuperuser Not Revoked

**Error:** User still has cloudsqlsuperuser role after creation

**Solutions:**
1. Ensure you're using the -gcp flag:
   ```bash
   go-create -s cloud-sql-host -u root -p password \
     --create-user myuser --create-pass "password..." \
     -r app_role -db mydb -g select -gcp
   ```

2. Manually revoke if needed:
   ```sql
   REVOKE cloudsqlsuperuser FROM 'myuser'@'%';
   ```

### Cloud SQL Connection Issues

**Error:** Cannot connect to Cloud SQL instance

**Solutions:**
1. Ensure Cloud SQL instance is running

2. Check authorized networks in Cloud SQL settings

3. Use Cloud SQL Proxy:
   ```bash
   cloud_sql_proxy -instances=PROJECT:REGION:INSTANCE=tcp:3306
   ```

4. Then connect via localhost:
   ```bash
   go-create -s localhost:3306 -u root -p password ...
   ```

## General Issues

### Config File Not Found

**Warning:** `Could not load config file`

**Solutions:**
1. Specify config file explicitly:
   ```bash
   go-create -config /path/to/config.json ...
   ```

2. Place config in default location:
   ```bash
   mkdir -p ~/.config
   cp config.json ~/.go-create.json
   ```

3. Use command-line flags instead:
   ```bash
   go-create -s hostname -u user -p password ...
   ```

### Temp Directory Issues

**Error:** `Failed to create temp directory`

**Solutions:**
1. Check /tmp directory exists and is writable:
   ```bash
   ls -ld /tmp
   chmod 1777 /tmp
   ```

2. Check disk space:
   ```bash
   df -h /tmp
   ```

3. Set TMPDIR environment variable:
   ```bash
   export TMPDIR=/path/to/writable/dir
   go-create ...
   ```

### Transaction Failed

**Error:** `Failed to commit transaction`

**Solutions:**
1. Check for conflicting operations

2. Retry the operation

3. Verify database server is not in read-only mode:
   ```sql
   SHOW VARIABLES LIKE 'read_only';
   ```

4. Check MySQL error log for details:
   ```bash
   tail -f /var/log/mysql/error.log
   ```

## Debugging Tips

### Enable Verbose Logging

Set environment variable:
```bash
export GO_CREATE_DEBUG=1
go-create ...
```

### Check MySQL Version

```bash
mysql -h hostname -u root -p -e "SELECT VERSION();"
```

### Verify User Creation

After creating a user, verify:
```bash
# Show user grants
go-create -show-user newuser

# Test connection
mysql -h hostname -u newuser -p
```

### SQL File Mode Debug

When using -use-sql-file, check the generated SQL:
```bash
# The tool will log the SQL file location
# Look for: "Created SQL file for user creation: /tmp/..."
```

### Test Connection

Use the built-in connection tester:
```bash
go-create -test-connection -user myuser -pass "password" -host hostname
```

## Getting Help

If you can't resolve your issue:

1. Check existing GitHub issues: https://github.com/ChaosHour/go-create/issues
2. Review the README.md for examples
3. Check SECURITY.md for security-related issues
4. Open a new issue with:
   - Go version: `go version`
   - MySQL version: `mysql --version`
   - Command you ran (mask passwords!)
   - Full error message
   - Operating system

## Common Patterns

### Creating Application User

```bash
# Development
go-create --create-user app_dev --create-pass "DevP@ssw0rd2024SecureAndLong!" \
  -r app_developer -g select,insert,update,delete -db dev_db

# Production
go-create --create-user app_prod --create-pass "ProdP@ssw0rd2024VerySecure&Long!" \
  -r app_readonly -g select -db prod_db
```

### Creating Read-Only User

```bash
go-create --create-user readonly --create-pass "ReadOnlyP@ss2024Secure&Long!" \
  -r read_only_role -g select -db analytics_db
```

### Creating Admin User

```bash
go-create --create-user dbadmin --create-pass "AdminP@ssw0rd2024SuperSecure!" \
  -r admin_role -g "all privileges" -db "*.*"
```

---

Last Updated: 2026-04-27
