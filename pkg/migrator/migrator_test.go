package migrator

import (
	"database/sql"
	"embed"
	"testing"

	"go.uber.org/zap"
)

func TestNewMigratorStoresDependencies(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	var fs embed.FS
	logger := zap.NewNop()

	migrator := NewMigrator(db, fs, logger)

	if migrator.db != db {
		t.Fatal("NewMigrator() did not store db")
	}
	if migrator.logg != logger {
		t.Fatal("NewMigrator() did not store logger")
	}
}
