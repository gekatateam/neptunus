# SQL drivers list

List of supported drivers and it's expected names:
 - [ClickHouse](https://github.com/ClickHouse/clickhouse-go/v2) as `clickhouse`
 - [MySQL](https://github.com/go-sql-driver/mysql) as `mysql`
 - [PostgreSQL](https://github.com/jackc/pgx/v4) as `pgx`, `postgres`
 - [SQL Server](https://github.com/microsoft/go-mssqldb) as `sqlserver`
 - [Oracle](https://github.com/sijms/go-ora/v2) as `oracle`, `ora`

> [!WARNING]  
> Oracle driver does NOT support queries canceling through Context passed to `ExecContext`, `QueryContext` and other similar methods. Please use [TIMEOUT](https://github.com/sijms/go-ora/?tab=readme-ov-file#timeout) connection option.
