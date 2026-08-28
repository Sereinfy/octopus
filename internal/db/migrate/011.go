package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{Version: 11, Up: migrateRemoveAutoGroupRegex})
}

func migrateRemoveAutoGroupRegex(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if db.Migrator().HasTable("channels") && hasPhysicalColumn(db, "channels", "auto_group") {
		if err := db.Exec("UPDATE channels SET auto_group = 0 WHERE auto_group IS NULL OR auto_group < 0 OR auto_group > 2").Error; err != nil {
			return fmt.Errorf("failed to normalize channels.auto_group: %w", err)
		}
	}
	if db.Migrator().HasTable("groups") {
		if err := dropColumnIfExists(db, &model.Group{}, "groups", "match_regex"); err != nil {
			return err
		}
	}
	return nil
}
