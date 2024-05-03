# GORM MySQL Driver

## Quick Start

```go
import (
  "gorm.io/driver/mysql"
  "gorm.io/gorm"
)

// https://github.com/go-sql-driver/mysql
dsn := "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local"
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
```

## Configuration

```go
import (
  "gorm.io/driver/mysql"
  "gorm.io/gorm"
)

var datetimePrecision = 2

db, err := gorm.Open(mysql.New(mysql.Config{
  DSN: "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local", // data source name, refer https://github.com/go-sql-driver/mysql#dsn-data-source-name
  DefaultStringSize: 256, // add default size for string fields, by default, will use db type `longtext` for fields without size, not a primary key, no index defined and don't have default values
  DisableDatetimePrecision: true, // disable datetime precision support, which not supported before MySQL 5.6
  DefaultDatetimePrecision: &datetimePrecision, // default datetime precision
  DontSupportRenameIndex: true, // drop & create index when rename index, rename index not supported before MySQL 5.7, MariaDB
  DontSupportRenameColumn: true, // use change when rename column, rename rename not supported before MySQL 8, MariaDB
  SkipInitializeWithVersion: false, // smart configure based on used version
}), &gorm.Config{})
```

## Customized Driver

```go
import (
  _ "example.com/my_mysql_driver"
  "gorm.io/gorm"
  "gorm.io/driver/mysql"
)

db, err := gorm.Open(mysql.New(mysql.Config{
  DriverName: "my_mysql_driver_name",
  DSN: "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local", // data source name, refer https://github.com/go-sql-driver/mysql#dsn-data-source-name
})
```

Checkout [https://gorm.io](https://gorm.io) for details.
