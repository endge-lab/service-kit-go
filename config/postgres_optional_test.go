package config

import "testing"

func TestPostgresRequiresExplicitDisable(t *testing.T) {
	c := ServiceConfig{}
	if c.validatePostgres() == nil {
		t.Fatal("missing PostgreSQL silently accepted")
	}
	disabled := false
	c.Postgres.Enabled = &disabled
	if err := c.validatePostgres(); err != nil {
		t.Fatal(err)
	}
	enabled := true
	c.Postgres.Enabled = &enabled
	if c.validatePostgres() == nil {
		t.Fatal("enabled PostgreSQL was not validated")
	}
}
