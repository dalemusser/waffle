# mysql

MySQL/MariaDB connection utilities for WAFFLE applications.

## Overview

The `mysql` package provides connection helpers for MySQL and MariaDB using Go's `database/sql` with timeout-bounded connections and connectivity verification.

## Import

```go
import "github.com/dalemusser/waffle/db/mysql"
```

---

## Connect

**Location:** `mysql.go`

```go
func Connect(dsn string, timeout time.Duration) (*sql.DB, error)
```

Opens a MySQL/MariaDB connection pool using Go's `database/sql`. The returned `*sql.DB` is a pool, not a single connection — it is safe for concurrent use and should be reused throughout the application.

**Example:**

```go
db, err := mysql.Connect("user:pass@tcp(localhost:3306)/mydb", 10*time.Second)
if err != nil {
    return err
}
defer db.Close()
```

---

## ConnectWithConfig

**Location:** `mysql.go`

```go
func ConnectWithConfig(dsn string, config PoolConfig, timeout time.Duration) (*sql.DB, error)
```

Opens a connection pool with custom pool settings.

**Example:**

```go
db, err := mysql.ConnectWithConfig(appCfg.MySQLDSN, mysql.PoolConfig{
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: time.Hour,
    ConnMaxIdleTime: 10 * time.Minute,
}, core.DBConnectTimeout)
```

---

## PoolConfig

**Location:** `mysql.go`

```go
type PoolConfig struct {
    MaxOpenConns    int           // Maximum open connections (0 = unlimited)
    MaxIdleConns    int           // Maximum idle connections (default: 2)
    ConnMaxLifetime time.Duration // Maximum connection reuse time (0 = unlimited)
    ConnMaxIdleTime time.Duration // Maximum idle time (0 = unlimited)
}
```

---

## DefaultPoolConfig

**Location:** `mysql.go`

```go
func DefaultPoolConfig() PoolConfig
```

Returns sensible defaults for production:

| Setting | Default |
|---------|---------|
| `MaxOpenConns` | 25 |
| `MaxIdleConns` | 5 |
| `ConnMaxLifetime` | 5 minutes |
| `ConnMaxIdleTime` | 5 minutes |

---

## WAFFLE Integration

### ConnectDB Hook

```go
// internal/app/bootstrap/db.go
func ConnectDB(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, logger *zap.Logger) (DBDeps, error) {
    db, err := mysql.ConnectWithConfig(appCfg.MySQLDSN, mysql.DefaultPoolConfig(), core.DBConnectTimeout)
    if err != nil {
        return DBDeps{}, fmt.Errorf("mysql: %w", err)
    }

    logger.Info("connected to MySQL")

    return DBDeps{DB: db}, nil
}
```

### Shutdown Hook

```go
func Shutdown(ctx context.Context, core *config.CoreConfig, appCfg AppConfig, db DBDeps, logger *zap.Logger) error {
    if db.DB != nil {
        if err := db.DB.Close(); err != nil {
            return fmt.Errorf("mysql close: %w", err)
        }
        logger.Info("disconnected from MySQL")
    }
    return nil
}
```

---

## Configuration

```go
type AppConfig struct {
    MySQLDSN string `conf:"mysql_dsn"`
}
```

```bash
# Environment variables
MYSQL_DSN=user:pass@tcp(localhost:3306)/mydb?parseTime=true
```

---

## DSN Format

```
# Basic format
user:password@tcp(host:port)/dbname

# With options (recommended)
user:password@tcp(host:port)/dbname?parseTime=true
user:password@tcp(host:port)/dbname?parseTime=true&loc=Local
user:password@tcp(host:port)/dbname?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci

# Unix socket
user:password@unix(/var/run/mysqld/mysqld.sock)/dbname
```

**Common DSN parameters:**

| Parameter | Description |
|-----------|-------------|
| `parseTime=true` | Parse `DATE` and `DATETIME` to `time.Time` (recommended) |
| `loc=Local` | Use local timezone for time parsing |
| `charset=utf8mb4` | Character set |
| `collation=utf8mb4_unicode_ci` | Collation |
| `timeout=10s` | Connection timeout |
| `readTimeout=30s` | Read timeout |
| `writeTimeout=30s` | Write timeout |

---

## See Also

- [app](../../app/app.md) — Application lifecycle hooks
- [config](../../config/config.md) — Core configuration including `DBConnectTimeout`
