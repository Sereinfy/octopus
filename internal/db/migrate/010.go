package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{Version: 10, Up: migrateAutoGroupFields})
}

func migrateAutoGroupFields(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if db.Migrator().HasTable("channels") && !hasPhysicalColumn(db, "channels", "auto_group") {
		if err := db.Exec("ALTER TABLE channels ADD COLUMN auto_group INTEGER NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("failed to add channels.auto_group: %w", err)
		}
	}
	if db.Migrator().HasTable("groups") && !hasPhysicalColumn(db, "groups", "match_regex") {
		if err := db.Exec("ALTER TABLE groups ADD COLUMN match_regex TEXT NOT NULL DEFAULT ''").Error; err != nil {
			return fmt.Errorf("failed to add groups.match_regex: %w", err)
		}
	}
	if db.Migrator().HasTable("group_items") {
		if !hasPhysicalColumn(db, "group_items", "source") {
			if err := db.Exec("ALTER TABLE group_items ADD COLUMN source TEXT NOT NULL DEFAULT 'manual'").Error; err != nil {
				return fmt.Errorf("failed to add group_items.source: %w", err)
			}
		}
		if err := db.Exec("UPDATE group_items SET source = 'manual' WHERE source IS NULL OR TRIM(source) = ''").Error; err != nil {
			return fmt.Errorf("failed to backfill group_items.source: %w", err)
		}
	}
	if db.Migrator().HasTable("channels") {
		if err := db.Exec("UPDATE channels SET auto_group = 0 WHERE auto_group IS NULL OR auto_group < 0 OR auto_group > 3").Error; err != nil {
			return fmt.Errorf("failed to normalize channels.auto_group: %w", err)
		}
	}
	return nil
}
