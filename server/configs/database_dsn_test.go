package configs

import (
	"net"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDatabaseConfigMySQLDSNRoundTrip(t *testing.T) {
	config := &DatabaseConfig{
		Dialect:  MySQL,
		Database: "team/archive +%?",
		Username: "worker+node",
		Password: "p@ss +/%':word",
		Host:     "2001:db8::1",
		Port:     3306,
		Params: map[string]string{
			"parseTime": "true",
			"sql_mode":  "ANSI_QUOTES,NO_BACKSLASH_ESCAPES",
		},
	}

	dsn, err := config.DSN()
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN: %v", err)
	}
	if parsed.User != config.Username {
		t.Errorf("user = %q, want %q", parsed.User, config.Username)
	}
	if parsed.Passwd != config.Password {
		t.Errorf("password did not round-trip")
	}
	if parsed.DBName != config.Database {
		t.Errorf("database = %q, want %q", parsed.DBName, config.Database)
	}
	if parsed.Net != "tcp" || parsed.Addr != net.JoinHostPort(config.Host, "3306") {
		t.Errorf("network address = %q(%q), want tcp(%q)", parsed.Net, parsed.Addr, net.JoinHostPort(config.Host, "3306"))
	}
	if !parsed.ParseTime {
		t.Error("parseTime parameter was not preserved")
	}
	if parsed.Params["sql_mode"] != config.Params["sql_mode"] {
		t.Errorf("sql_mode = %q, want %q", parsed.Params["sql_mode"], config.Params["sql_mode"])
	}
}

func TestDatabaseConfigMySQLDSNRejectsColonInUsername(t *testing.T) {
	config := &DatabaseConfig{
		Dialect:  MySQL,
		Username: "worker:node",
		Password: "password",
		Host:     "localhost",
		Port:     3306,
	}
	if _, err := config.DSN(); err == nil {
		t.Fatal("expected an error for a username that the MySQL DSN grammar cannot represent")
	}
}

func TestDatabaseConfigPostgresDSNRoundTrip(t *testing.T) {
	config := &DatabaseConfig{
		Dialect:  Postgres,
		Database: "team db/primary'\\",
		Username: "worker's account\\ops",
		Password: "secret 'with' spaces\\and/slashes",
		Host:     "db.internal",
		Port:     5432,
		Params: map[string]string{
			"application_name": "audit worker's \\ trial",
			"search_path":      "tenant, public",
			"sslmode":          "disable",
		},
	}

	dsn, err := config.DSN()
	if err != nil {
		t.Fatalf("DSN: %v", err)
	}
	parsed, err := pgconn.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("pgconn.ParseConfig: %v", err)
	}
	if parsed.User != config.Username {
		t.Errorf("user = %q, want %q", parsed.User, config.Username)
	}
	if parsed.Password != config.Password {
		t.Errorf("password did not round-trip")
	}
	if parsed.Database != config.Database {
		t.Errorf("database = %q, want %q", parsed.Database, config.Database)
	}
	if parsed.Host != config.Host || parsed.Port != config.Port {
		t.Errorf("address = %q:%d, want %q:%d", parsed.Host, parsed.Port, config.Host, config.Port)
	}
	for _, key := range []string{"application_name", "search_path"} {
		if parsed.RuntimeParams[key] != config.Params[key] {
			t.Errorf("%s = %q, want %q", key, parsed.RuntimeParams[key], config.Params[key])
		}
	}
	if parsed.TLSConfig != nil {
		t.Error("sslmode=disable was not preserved")
	}
}

func TestDatabaseConfigPostgresDSNRejectsInvalidParameterKey(t *testing.T) {
	config := &DatabaseConfig{
		Dialect:  Postgres,
		Database: "sliver",
		Username: "worker",
		Host:     "db.internal",
		Port:     5432,
		Params: map[string]string{
			"sslmode=disable user": "attacker",
		},
	}
	if _, err := config.DSN(); err == nil {
		t.Fatal("expected an error for a PostgreSQL parameter key that can inject another keyword")
	}
}
