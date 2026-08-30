package migrate

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateRepairGroupItemChannelModel(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:repair-group-item-channel-model?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		"CREATE TABLE channels (id INTEGER PRIMARY KEY, auto_sync INTEGER NOT NULL DEFAULT 1, model TEXT, custom_model TEXT)",
		"CREATE TABLE groups (id INTEGER PRIMARY KEY, active_item_id INTEGER NOT NULL DEFAULT 0)",
		"CREATE TABLE group_items (id INTEGER PRIMARY KEY, group_id INTEGER NOT NULL, channel_id INTEGER NOT NULL, model_name TEXT NOT NULL, priority INTEGER NOT NULL)",
		"CREATE UNIQUE INDEX idx_group_channel_model ON group_items (group_id, channel_id, model_name)",
		"INSERT INTO channels (id, auto_sync, model, custom_model) VALUES (1, 1, 'gpt-4', 'manual-model')",
		"INSERT INTO groups (id, active_item_id) VALUES (1, 1)",
		"INSERT INTO groups (id, active_item_id) VALUES (2, 2)",
		"INSERT INTO group_items (id, group_id, channel_id, model_name, priority) VALUES (1, 1, 1, 'gpt-4', 7)",
		"INSERT INTO group_items (id, group_id, channel_id, model_name, priority) VALUES (2, 1, 999, 'orphan', 8)",
	}
	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateRepairGroupItemChannelModel(database); err != nil {
		t.Fatal(err)
	}

	if !hasPhysicalColumn(database, "group_items", "channel_model_id") {
		t.Fatal("group_items.channel_model_id was not added")
	}
	if hasPhysicalColumn(database, "group_items", "channel_id") || hasPhysicalColumn(database, "group_items", "model_name") {
		t.Fatal("legacy group_items columns were not removed")
	}

	var channelModel model.ChannelModel
	if err := database.First(&channelModel, "channel_id = ? AND name = ?", 1, "gpt-4").Error; err != nil {
		t.Fatal(err)
	}
	if channelModel.ID == 0 {
		t.Fatal("channel model id was not assigned")
	}

	var items []model.GroupItem
	if err := database.Order("id ASC").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 1 || items[0].ChannelModelID != channelModel.ID || items[0].Priority != 7 {
		t.Fatalf("unexpected migrated items: %+v", items)
	}

	var activeItemID int
	if err := database.Table("groups").Select("active_item_id").Where("id = 1").Scan(&activeItemID).Error; err != nil {
		t.Fatal(err)
	}
	if activeItemID != 1 {
		t.Fatalf("valid active item was cleared: %d", activeItemID)
	}
	var staleActiveItemID int
	if err := database.Table("groups").Select("active_item_id").Where("id = 2").Scan(&staleActiveItemID).Error; err != nil {
		t.Fatal(err)
	}
	if staleActiveItemID != 0 {
		t.Fatalf("stale active item was not cleared: %d", staleActiveItemID)
	}
}
