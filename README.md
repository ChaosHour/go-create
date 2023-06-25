# go-create


## Description:
This tool is used to create users, roles, and grants in MySQL.  

## !!!!WARNING!!!!
This is only used currently for testing. Do not use in PROD or any evirement that you care about. 
More testing and validation needs to happen before this is ready for PROD.


## Usage:
```GO
go-create -h
  -db string
        Database name
  -g string
        Comma-separated list of grants to create
  -h    Print help
  -p string
        Password
  -r string
        Comma-separated list of roles to create
  -s string
        Source Host
  -u string
        User
```

## Example:
```GO
Passwords created for testing with:
  pwgen -s -c -n 23 1

Database created for testing with:
  mysqladmin create app_db  

❯ go-create -s 10.8.0.15 -u lisa -p OxFF29szWNQ962hUa0Toez3 -r app_write -g select,insert,update,delete -db app_db 
2023/06/25 10:54:13 [+] Connecting to database: root:root@tcp(10.8.0.15:3306)/mysql
2023/06/25 10:54:13 [!] Role app_write already exists
2023/06/25 10:54:13 [+] Granted privileges to role: app_write
2023/06/25 10:54:13 [+] Created user: lisa
2023/06/25 10:54:13 [+] Granted role to user: lisa
2023/06/25 10:54:13 [+] Granted privileges to user: lisa
2023/06/25 10:54:13 [+] Set default role for user: lisa
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

## Using go-pass to validate:
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
