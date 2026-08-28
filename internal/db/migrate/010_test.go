package migrate

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateAutoGroupFieldsAddsDefaults(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:auto-group-migration-add?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		"CREATE TABLE channels (id INTEGER PRIMARY KEY)",
		"CREATE TABLE groups (id INTEGER PRIMARY KEY, name TEXT NOT NULL)",
		"CREATE TABLE group_items (id INTEGER PRIMARY KEY, group_id INTEGER NOT NULL)",
		"INSERT INTO channels (id) VALUES (1)",
		"INSERT INTO groups (id, name) VALUES (1, 'gpt-4')",
		"INSERT INTO group_items (id, group_id) VALUES (1, 1)",
	}
	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateAutoGroupFields(database); err != nil {
		t.Fatal(err)
	}
	for table, column := range map[string]string{
		"channels":    "auto_group",
		"groups":      "match_regex",
		"group_items": "source",
	} {
		if !hasPhysicalColumn(database, table, column) {
			t.Fatalf("%s.%s was not added", table, column)
		}
	}
	var autoGroup int
	if err := database.Raw("SELECT auto_group FROM channels WHERE id = 1").Scan(&autoGroup).Error; err != nil || autoGroup != 0 {
		t.Fatalf("unexpected auto_group default: %d, %v", autoGroup, err)
	}
	var source string
	if err := database.Raw("SELECT source FROM group_items WHERE id = 1").Scan(&source).Error; err != nil || source != "manual" {
		t.Fatalf("unexpected source default: %q, %v", source, err)
	}
	var matchRegex string
	if err := database.Raw("SELECT match_regex FROM groups WHERE id = 1").Scan(&matchRegex).Error; err != nil || matchRegex != "" {
		t.Fatalf("unexpected match_regex default: %q, %v", matchRegex, err)
	}
}

func TestMigrateAutoGroupFieldsNormalizesInvalidValues(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:auto-group-migration-normalize?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		"CREATE TABLE channels (id INTEGER PRIMARY KEY, auto_group INTEGER)",
		"CREATE TABLE groups (id INTEGER PRIMARY KEY, match_regex TEXT)",
		"CREATE TABLE group_items (id INTEGER PRIMARY KEY, source TEXT)",
		"INSERT INTO channels (id, auto_group) VALUES (1, 99)",
		"INSERT INTO group_items (id, source) VALUES (1, '')",
	}
	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateAutoGroupFields(database); err != nil {
		t.Fatal(err)
	}
	var autoGroup int
	if err := database.Raw("SELECT auto_group FROM channels WHERE id = 1").Scan(&autoGroup).Error; err != nil || autoGroup != 0 {
		t.Fatalf("invalid auto_group was not reset: %d, %v", autoGroup, err)
	}
	var source string
	if err := database.Raw("SELECT source FROM group_items WHERE id = 1").Scan(&source).Error; err != nil || source != "manual" {
		t.Fatalf("empty source was not backfilled: %q, %v", source, err)
	}
}
