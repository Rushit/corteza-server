package rules_test

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/namsral/flag"
	"github.com/titpetric/factory"

	systemMigrate "github.com/crusttech/crust/system/db"
)

func TestMain(m *testing.M) {
	// @todo this is a very optimistic initialization, make it more robust
	godotenv.Load("../../.env")

	prefix := "system"
	dsn := ""

	p := func(s string) string {
		return prefix + "-" + s
	}

	flag.StringVar(&dsn, p("db-dsn"), "crust:crust@tcp(db1:3306)/crust?collation=utf8mb4_general_ci", "DSN for database connection")
	flag.Parse()

	factory.Database.Add("default", dsn)

	db := factory.Database.MustGet()
	db.Profiler = &factory.Database.ProfilerStdout

	// migrate database schema
	if err := systemMigrate.Migrate(db); err != nil {
		log.Printf("Error running migrations: %+v\n", err)
		return
	}

	// clean up tables
	{
		for _, name := range []string{"sys_user", "sys_role", "sys_role_member", "sys_organisation", "sys_rules"} {
			_, err := db.Exec("truncate " + name)
			if err != nil {
				panic("Error when clearing " + name + ": " + err.Error())
			}
		}
	}

	os.Exit(m.Run())
}