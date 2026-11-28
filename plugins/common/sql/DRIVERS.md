# SQL drivers list

List of supported drivers and it's expected names:
 - [ClickHouse](https://github.com/ClickHouse/clickhouse-go/v2) as `clickhouse`
 - [MySQL](https://github.com/go-sql-driver/mysql) as `mysql`
 - [PostgreSQL](https://github.com/jackc/pgx/v5) as `pgx`, `postgres`
 - [SQL Server](https://github.com/microsoft/go-mssqldb) as `sqlserver`
 - [Oracle](https://github.com/sijms/go-ora/v3) as `oracle`, `ora`
 - [Oracle < 19c](https://github.com/sijms/go-ora/v2) as `oraclelegacy`, `oralegacy` (see [this](https://github.com/sijms/go-ora/issues/759) issue to more details)

> [!WARNING]  
> Oracle driver has some issues with queries canceling through Context passed to `ExecContext`, `QueryContext` and other similar methods. Please use [TIMEOUT/CONNECT TIMEOUT](https://github.com/sijms/go-ora/tree/master/v3#connection-options) connection option.
