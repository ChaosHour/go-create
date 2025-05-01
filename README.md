# go-create

## Description

This tool is used to create users, roles, and grants in MySQL. It features robust handling of complex passwords and role-based access control.

## WARNING

This is only used currently for testing. Do not use in PROD or any environment that you care about.
More testing and validation needs to happen before this is ready for PROD.

## Configuration

You can store your MySQL connection details in a configuration file. By default, the tool looks for `.go-create.json` in your home directory, or you can specify a different path using the `-config` flag.

Example configuration file:

```json
{
  "mysql": {
    "host": "192.168.50.50",
    "port": "3306",
    "user": "root",
    "password": "your_password"
  }
}
```

Credentials precedence:

1. Command line flags (-u, -p, -s)
2. Configuration file specified by -config
3. Default .go-create.json in home directory
4. ~/.my.cnf file

## Usage

```GO
go-create -h
  -config string
        Path to configuration file
  -create-user string
        Username to create/modify
  -create-pass string
        Password for the user being created
  -db string
        Database name
  -g string
        Comma-separated list of grants to create
  -gcp
        Revoke cloudsqlsuperuser role after granting roles (for GCP Cloud SQL)
  -h    
        Print help
  -p string
        Password for the admin connection
  -r string
        Comma-separated list of roles to create
  -s string
        Source Host (MySQL server address)
  -show
        Show grants for specified role (requires -r flag)
  -show-user string
        Show grants for the specified username
  -u string
        Username for the admin connection
```

## Examples

### 1. Creating a user with role and privileges

```GO
# Create user 'lisa' with role 'app_write' and specific database privileges
go-create -s 10.8.0.15 -u lisa -p OxFF29szWNQ962hUa0Toez3 -r app_write -g select,insert,update,delete -db app_db 
```

### 2. Creating a role with privileges

```GO
# Create role 'app_read2' with SELECT privilege on app_db
go-create -s 10.8.0.15 -r app_read2 -g select -db app_db
```

### 3. Showing user grants

```GO
# Show grants for user 'lisa'
go-create -show-user lisa
```

### 4. Creating a user with role in Google Cloud SQL

```GO
# Create user 'repl' with role 'repl_role', specific database privileges, and revoke cloudsqlsuperuser
# Note: -u and -p are for admin credentials, --create-user and --create-pass are for the new user
go-create -s cloud-sql-instance -u root -p s3cr3t --create-user repl --create-pass replpass -g select,insert,update,delete -db app_db -r repl_role -gcp

# The above command will:
# 1. Connect as root to create the new user
# 2. Create user 'repl' with password 'replpass'
# 3. Create role 'repl_role'
# 4. Grant the specified privileges to the role
# 5. Grant the role to the user
# 6. Revoke cloudsqlsuperuser role (due to -gcp flag)
```

Note: The -gcp flag specifically handles Google Cloud SQL instances where users are automatically granted the 'cloudsqlsuperuser' role. When this flag is used, the tool will automatically revoke this role after granting the specified roles.

### 5. Creating a user with role in Google Cloud SQL

```GO
# Create user 'repl' with role 'repl_role', specific database privileges, and revoke cloudsqlsuperuser
go-create -s cloud-sql-instance -u root -p s3cr3t --create-user repl --create-pass replpass -g select,insert,update,delete -db app_db -r repl_role -gcp

# Output will include:
# [+] Created user: repl@%
# [+] Created role: repl_role
# [+] Granted privileges to role: repl_role
# [+] Granted role to user: repl
# [+] Revoked cloudsqlsuperuser role from user: repl@%
```

Note: When using with Google Cloud SQL:

- The `-u` and `-p` flags are for the admin credentials (to connect to the database)
- The `--create-user` and `--create-pass` flags specify the new user to create
- The `-gcp` flag ensures the cloudsqlsuperuser role is revoked after granting the specified roles

### 6. Using credentials from .my.cnf

```GO
# Create role 'app_read' with SELECT privilege on chaos database
# Note: No -u/-p/-s flags needed when using .my.cnf
go-create -r app_read -g select -db chaos

# Output will include:
# [+] Using credentials from .my.cnf
# [+] Connecting to MySQL server at 192.168.50.50:3306 (using .my.cnf)
# [+] Created role: app_read
# [+] Granted privileges to role: app_read
```

### 7. Show role privileges

```GO
# Show privileges for role 'app_read'
go-create -r app_read -show

# Output:
2025/02/17 10:33:55 [+] Using credentials from .my.cnf
2025/02/17 10:33:55 [+] Connecting to MySQL server at 192.168.50.50:3306 (using .my.cnf)
2025/02/17 10:33:55 [+] Grants for role app_read:
2025/02/17 10:33:55     GRANT USAGE ON *.* TO `app_read`@`%`
2025/02/17 10:33:55     GRANT SELECT ON `chaos`.* TO `app_read`@`%`
```

### 8. Creating a user with password, role and privileges

```GO
# Create user 'lisa' with password, role 'app_write', and specific database privileges
go-create --create-user lisa --create-pass OxFF29szWNQ962hUa0Toez3 -r app_write -g select,insert,update,delete -db chaos

# Output:
2025/02/17 10:36:46 [+] Using credentials from .my.cnf
2025/02/17 10:36:46 [+] Connecting to MySQL server at 192.168.50.50:3306 (using .my.cnf)
2025/02/17 10:36:46 [+] Created role: app_write
2025/02/17 10:36:46 [+] Granted privileges to role: app_write
2025/02/17 10:36:46 [+] Created user: lisa@%
2025/02/17 10:36:46 [+] Granted role to user: lisa
2025/02/17 10:36:46 [+] Granted privileges to user: lisa
2025/02/17 10:36:46 [+] Set default role for user: lisa

# Verify role privileges:
go-create -r app_write -show
2025/02/17 10:37:23 [+] Using credentials from .my.cnf
2025/02/17 10:37:23 [+] Connecting to MySQL server at 192.168.50.50:3306 (using .my.cnf)
2025/02/17 10:37:23 [+] Grants for role app_write:
2025/02/17 10:37:23     GRANT USAGE ON *.* TO `app_write`@`%`
2025/02/17 10:37:23     GRANT SELECT, INSERT, UPDATE, DELETE ON `chaos`.* TO `app_write`@`%`

# Verify user grants:
go-create -show-user lisa
2025/02/17 10:38:10 [+] Using credentials from .my.cnf
2025/02/17 10:38:10 [+] Connecting to MySQL server at 192.168.50.50:3306 (using .my.cnf)
2025/02/17 10:38:10 [+] Grants for user lisa:
2025/02/17 10:38:10     GRANT USAGE ON *.* TO `lisa`@`%`
2025/02/17 10:38:10     GRANT SELECT, INSERT, UPDATE, DELETE ON `chaos`.* TO `lisa`@`%`
2025/02/17 10:38:10     GRANT `app_write`@`%` TO `lisa`@`%`
```

## Testing the Connection After User Creation

After creating a user, you can test the connection using the MySQL client:

```sh
mysql -h <host> -u <new_user> -p
```
Enter the password when prompted.

Or, using the password file created earlier:

```sh
mysql -h <host> -u <new_user> -p"$(< /path/to/password_file.txt)"
```

Alternatively, you can use the `go-create` tool's built-in connection test:

```sh
go-create -test-connection -user <new_user> -pass "<password>" -host <host>
```

## Validation

```GO
❯ mysql -vv -e "source test.sql" 
--------------
show databases
--------------

+--------------------+
| Database           |
+--------------------+
| app_db             |
| information_schema |
| mysql              |
| performance_schema |
| sys                |
+--------------------+
5 rows in set (0.00 sec)

--------------
Select user,host from mysql.user where account_locked ='Y' and password_expired='Y' order by 1
--------------

+------------+------+
| user       | host |
+------------+------+
| app_write  | %    |
| app_write2 | %    |
| read_only  | %    |
+------------+------+
3 rows in set (0.01 sec)

--------------
SELECT user AS role_name FROM mysql.user WHERE host = '%' AND NOT LENGTH(authentication_string)
--------------

+------------+
| role_name  |
+------------+
| app_write  |
| app_write2 |
| read_only  |
+------------+
3 rows in set (0.00 sec)

--------------
SELECT DISTINCT User 'Role Name', if(from_user is NULL,0, 1) Active FROM mysql.user LEFT JOIN role_edges ON from_user=user WHERE account_locked='Y' AND password_expired='Y' AND authentication_string=''
--------------

+------------+--------+
| Role Name  | Active |
+------------+--------+
| app_write  |      1 |
| app_write2 |      0 |
| read_only  |      1 |
+------------+--------+
3 rows in set (0.00 sec)

--------------
SELECT * FROM mysql.role_edges
--------------

+-----------+-----------+---------+---------+-------------------+
| FROM_HOST | FROM_USER | TO_HOST | TO_USER | WITH_ADMIN_OPTION |
+-----------+-----------+---------+---------+-------------------+
| %         | app_write | %       | lisa    | N                 |
| %         | read_only | %       | klarsen | N                 |
+-----------+-----------+---------+---------+-------------------+
2 rows in set (0.00 sec)

--------------
select * from information_schema.user_privileges where GRANTEE='\'mysql.infoschema\'@\'localhost\''
--------------

+--------------------------------+---------------+--------------------+--------------+
| GRANTEE                        | TABLE_CATALOG | PRIVILEGE_TYPE     | IS_GRANTABLE |
+--------------------------------+---------------+--------------------+--------------+
| 'mysql.infoschema'@'localhost' | def           | SELECT             | NO           |
| 'mysql.infoschema'@'localhost' | def           | SYSTEM_USER        | NO           |
| 'mysql.infoschema'@'localhost' | def           | FIREWALL_EXEMPT    | NO           |
| 'mysql.infoschema'@'localhost' | def           | AUDIT_ABORT_EXEMPT | NO           |
+--------------------------------+---------------+--------------------+--------------+
4 rows in set (0.01 sec)

--------------
SELECT user,host FROM mysql.user
--------------

+------------------+-----------+
| user             | host      |
+------------------+-----------+
| app_write        | %         |
| app_write2       | %         |
| chaoshour        | %         |
| johnny5          | %         |
| klarsen          | %         |
| lisa             | %         |
| read_only        | %         |
| root             | %         |
| mysql.infoschema | localhost |
| mysql.session    | localhost |
| mysql.sys        | localhost |
| root             | localhost |
+------------------+-----------+
12 rows in set (0.00 sec)

Bye
```

## Using go-pass to validate

- [go-pass](https://github.com/ChaosHour/go-pass)

```GO
go-pass -s 10.8.0.15 -f show_users.sql -o lisa | sed -e 's/CREATE USER/CREATE USER IF NOT EXISTS/g' -e '/^-- Grants/d' | grep -v 'Dumping' > only-lisa.sql
2023/06/25 10:59:42 [+] Connecting to database: root:root@tcp(10.8.0.15:3306)/mysql

cat only-lisa.sql
-- CREATE USER IF NOT EXISTS for lisa@%:
 CREATE USER IF NOT EXISTS `lisa`@`%` IDENTIFIED WITH 'caching_sha2_password' AS 0x244124303035246B373E322A6C59350A3641206C742E26402B7A2A55726B473177614C6E5A4D71586E55612E776D5937376445434454744A722F76426F67304D4B54686C2E32 DEFAULT ROLE `app_write`@`%` REQUIRE NONE PASSWORD EXPIRE DEFAULT ACCOUNT UNLOCK PASSWORD HISTORY DEFAULT PASSWORD REUSE INTERVAL DEFAULT PASSWORD REQUIRE CURRENT DEFAULT;
 GRANT USAGE ON *.* TO `lisa`@`%`;
 GRANT SELECT, INSERT, UPDATE, DELETE ON `app_db`.* TO `lisa`@`%`;
 GRANT `app_write`@`%` TO `lisa`@`%`;
```

## Example - 2

```GO
Create a new role called app_read2 with grants and db:

go-create on  main via 🐹 v1.20.5 
❯ go-create -s 10.8.0.15 -r app_read2 -g select -db app_db                          
2023/06/25 15:22:50 [+] Connecting to database: root:root@tcp(10.8.0.15:3306)/mysql
2023/06/25 15:22:50 [+] Created role: app_read2
2023/06/25 15:22:50 [+] Granted privileges to role: app_read2


Create new user lisa3 and add lisa3 to a default role of app_read2:

go-create on  main via 🐹 v1.20.5 
❯ go-create -s 10.8.0.15 -u lisa3 -p OxFF29szWNQ962hUa0Toez3 -r app_read2            
2023/06/25 15:23:37 [+] Connecting to database: root:root@tcp(10.8.0.15:3306)/mysql
2023/06/25 15:23:37 [!] Role app_read2 already exists
2023/06/25 15:23:37 [+] Created user: lisa3
2023/06/25 15:23:37 [+] Granted role to user: lisa3
2023/06/25 15:23:37 [+] Set default role for user: lisa3



Connect to MySQL 8 and show grants:

go-create on  main via 🐹 v1.20.5 
❯ mysql -u lisa3 -pOxFF29szWNQ962hUa0Toez3

Welcome to the MySQL monitor.  Commands end with ; or \g.
Your MySQL connection id is 216
Server version: 8.0.32-24 Percona Server (GPL), Release 24, Revision e5c6e9d2

Copyright (c) 2000, 2023, Oracle and/or its affiliates.

Oracle is a registered trademark of Oracle Corporation and/or its
affiliates. Other names may be trademarks of their respective
owners.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

mysql> show grants;
+-------------------------------------------+
| Grants for lisa3@%                        |
+-------------------------------------------+
| GRANT USAGE ON *.* TO `lisa3`@`%`         |
| GRANT SELECT ON `app_db`.* TO `lisa3`@`%` |
| GRANT `app_read2`@`%` TO `lisa3`@`%`      |
+-------------------------------------------+
3 rows in set (0.00 sec)

```

## Example - 3

```GO
Show grants for an existing user:

❯ go-create -show-user lisa
2023/06/25 15:24:50 [+] Connecting to database: root:root@tcp(10.8.0.15:3306)/mysql
2023/06/25 15:24:50 [+] Grants for user lisa:
    GRANT USAGE ON *.* TO `lisa`@`%`
    GRANT SELECT, INSERT, UPDATE, DELETE ON `app_db`.* TO `lisa`@`%`
    GRANT `app_write`@`%` TO `lisa`@`%`
