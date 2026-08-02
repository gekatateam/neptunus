# SQL drivers list

List of supported drivers and it's expected names:
 - [ClickHouse](https://github.com/ClickHouse/clickhouse-go/v2) as `clickhouse`
 - [MySQL](https://github.com/go-sql-driver/mysql) as `mysql`
 - [PostgreSQL](https://github.com/jackc/pgx/v5) as `pgx`, `postgres`
 - [SQL Server](https://github.com/microsoft/go-mssqldb) as `sqlserver`
 - [Oracle](https://github.com/sijms/go-ora/v3) as `oracle`, `ora`

> [!WARNING]  
> Oracle driver has some issues with queries canceling through Context passed to `ExecContext`, `QueryContext` and other similar methods. Please use [TIMEOUT/CONNECT TIMEOUT](https://github.com/sijms/go-ora/tree/master/v3#connection-options) connection option.
