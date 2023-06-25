\u mysql

show databases;

Select user,host from mysql.user where account_locked ='Y' and password_expired='Y' order by 1;

SELECT user AS role_name FROM mysql.user WHERE host = '%' AND NOT LENGTH(authentication_string);

SELECT DISTINCT User 'Role Name', if(from_user is NULL,0, 1) Active FROM mysql.user LEFT JOIN role_edges ON from_user=user WHERE account_locked='Y' AND password_expired='Y' AND authentication_string='';

SELECT * FROM mysql.role_edges;

select * from information_schema.user_privileges where GRANTEE='\'mysql.infoschema\'@\'localhost\'';

SELECT user,host FROM mysql.user;