package server_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rera/property-platform/backend/internal/config"
)

// testSchema is the schema this process owns. Two `go test` runs against the
// same machine would otherwise share one schema: each one's setup() truncates
// the other's rows mid-test and each one's AutoMigrate alters tables the other
// has open. The symptoms are not obviously about isolation — a create that
// answers 404 because its rows vanished, a register that fails with "Table
// definition has changed" — and, worst of all, they land on a different test
// every run.
//
// Naming the schema after the process makes concurrent runs impossible to
// confuse and costs one CREATE and one DROP per run.
var testSchema = fmt.Sprintf("%s_%d", envOr("TEST_DB_NAME", "rera_properties_test"), os.Getpid())

func TestMain(m *testing.M) {
	code := m.Run()
	dropTestSchema()
	os.Exit(code)
}

// dropTestSchema removes this process's schema. A run killed part-way leaves
// its schema behind; those are safe to drop by hand.
func dropTestSchema() {
	config.LoadDotEnv()
	cfg := config.Load()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return
	}
	defer db.Close()
	_, _ = db.Exec("DROP DATABASE IF EXISTS `" + testSchema + "`")
}
