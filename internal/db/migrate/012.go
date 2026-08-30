package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	// 版本 10 已被本项目用于自动分组字段，故使用新的版本号承载上游的旧库修复。
	RegisterBeforeAutoMigration(Migration{
		Version: 12,
		Up:      migrateRepairGroupItemChannelModel,
	})
}

// migrateRepairGroupItemChannelModel 修补版本 8 未执行完成的数据库。
// group_items 仍保留 channel_id 和 model_name 时，补齐渠道模型记录，
// 就地回填 channel_model_id 并删除旧列，保留原有分组项主键和顺序。
func migrateRepairGroupItemChannelModel(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("group_items") ||
		!hasPhysicalColumn(db, "group_items", "channel_id") ||
		!hasPhysicalColumn(db, "group_items", "model_name") {
		return nil
	}
	if !db.Migrator().HasTable("channels") {
		return fmt.Errorf("channels table not found")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if !tx.Migrator().HasTable("channel_models") {
			if err := tx.AutoMigrate(&model.ChannelModel{}); err != nil {
				return fmt.Errorf("failed to create channel_models: %w", err)
			}
		}

		type legacyGroupItem struct {
			ID        int
			ChannelID int
			ModelName string
			GroupID   int
			Priority  int
		}
		legacyItems := make([]legacyGroupItem, 0)
		groupItemFields := []string{"id", "channel_id", "model_name", "group_id"}
		if hasPhysicalColumn(tx, "group_items", "priority") {
			groupItemFields = append(groupItemFields, "priority")
		}
		if err := tx.Table("group_items").
			Select(strings.Join(groupItemFields, ", ")).
			Order("id ASC").
			Find(&legacyItems).Error; err != nil {
			return fmt.Errorf("failed to read legacy group_items: %w", err)
		}

		type legacyChannel struct {
			ID          int
			AutoSync    bool
			Models      sql.NullString `gorm:"column:model"`
			CustomModel sql.NullString `gorm:"column:custom_model"`
		}
		channelFields := []string{"id"}
		if hasPhysicalColumn(tx, "channels", "auto_sync") {
			channelFields = append(channelFields, "auto_sync")
		}
		if hasPhysicalColumn(tx, "channels", "model") {
			channelFields = append(channelFields, "model")
		}
		if hasPhysicalColumn(tx, "channels", "custom_model") {
			channelFields = append(channelFields, "custom_model")
		}
		channels := make([]legacyChannel, 0)
		if err := tx.Table("channels").Select(channelFields).Order("id ASC").Find(&channels).Error; err != nil {
			return fmt.Errorf("failed to read legacy channels: %w", err)
		}
		channelAutoSync := make(map[int]bool, len(channels))
		for _, channel := range channels {
			channelAutoSync[channel.ID] = channel.AutoSync
		}

		type channelModelIndexKey struct {
			channelID int
			name      string
		}
		channelModels := make([]model.ChannelModel, 0)
		if err := tx.Order("id ASC").Find(&channelModels).Error; err != nil {
			return fmt.Errorf("failed to read channel_models: %w", err)
		}
		idByName := make(map[channelModelIndexKey]int, len(channelModels))
		idByLowerName := make(map[channelModelIndexKey]int, len(channelModels))
		indexChannelModels := func() {
			idByName = make(map[channelModelIndexKey]int, len(channelModels))
			idByLowerName = make(map[channelModelIndexKey]int, len(channelModels))
			for _, channelModel := range channelModels {
				name := strings.TrimSpace(channelModel.Name)
				exact := channelModelIndexKey{channelID: channelModel.ChannelID, name: name}
				if _, ok := idByName[exact]; !ok {
					idByName[exact] = channelModel.ID
				}
				lower := channelModelIndexKey{channelID: channelModel.ChannelID, name: strings.ToLower(name)}
				if _, ok := idByLowerName[lower]; !ok {
					idByLowerName[lower] = channelModel.ID
				}
			}
		}
		indexChannelModels()
		findModelID := func(channelID int, name string) (int, bool) {
			name = strings.TrimSpace(name)
			if id, ok := idByName[channelModelIndexKey{channelID: channelID, name: name}]; ok {
				return id, true
			}
			id, ok := idByLowerName[channelModelIndexKey{channelID: channelID, name: strings.ToLower(name)}]
			return id, ok
		}

		missing := make([]model.ChannelModel, 0)
		missingIndexByKey := make(map[channelModelIndexKey]int)
		addModel := func(channelID int, name string, source model.ChannelModelSource) {
			name = strings.TrimSpace(name)
			if name == "" {
				return
			}
			if _, ok := channelAutoSync[channelID]; !ok {
				return
			}
			if _, ok := findModelID(channelID, name); ok {
				return
			}
			key := channelModelIndexKey{channelID: channelID, name: name}
			if index, ok := missingIndexByKey[key]; ok {
				if source == model.ChannelModelSourceManual {
					missing[index].Source = model.ChannelModelSourceManual
				}
				return
			}
			missingIndexByKey[key] = len(missing)
			missing = append(missing, model.ChannelModel{ChannelID: channelID, Name: name, Source: source})
		}
		for _, channel := range channels {
			if channel.Models.Valid {
				for _, name := range strings.Split(channel.Models.String, ",") {
					addModel(channel.ID, name, model.ChannelModelSourceAuto)
				}
			}
			if channel.CustomModel.Valid {
				for _, name := range strings.Split(channel.CustomModel.String, ",") {
					addModel(channel.ID, name, model.ChannelModelSourceManual)
				}
			}
		}
		for _, item := range legacyItems {
			source := model.ChannelModelSourceManual
			if channelAutoSync[item.ChannelID] {
				source = model.ChannelModelSourceAuto
			}
			addModel(item.ChannelID, item.ModelName, source)
		}
		if len(missing) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&missing).Error; err != nil {
				return fmt.Errorf("failed to create channel_models: %w", err)
			}
			channelModels = channelModels[:0]
			if err := tx.Order("id ASC").Find(&channelModels).Error; err != nil {
				return fmt.Errorf("failed to reload channel_models: %w", err)
			}
			indexChannelModels()
		}

		if !hasPhysicalColumn(tx, "group_items", "channel_model_id") {
			if err := tx.Exec("ALTER TABLE group_items ADD COLUMN channel_model_id integer NOT NULL DEFAULT 0").Error; err != nil {
				return fmt.Errorf("failed to add group_items.channel_model_id: %w", err)
			}
		}

		invalidIDs := make([]int, 0)
		seen := make(map[[2]int]struct{}, len(legacyItems))
		for _, item := range legacyItems {
			channelModelID, ok := findModelID(item.ChannelID, item.ModelName)
			itemKey := [2]int{item.GroupID, channelModelID}
			if !ok || channelModelID == 0 {
				invalidIDs = append(invalidIDs, item.ID)
				continue
			}
			if _, exists := seen[itemKey]; exists {
				invalidIDs = append(invalidIDs, item.ID)
				continue
			}
			seen[itemKey] = struct{}{}
			if err := tx.Model(&model.GroupItem{}).Where("id = ?", item.ID).
				Update("channel_model_id", channelModelID).Error; err != nil {
				return fmt.Errorf("failed to update group_item %d: %w", item.ID, err)
			}
		}
		if len(invalidIDs) > 0 {
			if err := tx.Where("id IN ?", invalidIDs).Delete(&model.GroupItem{}).Error; err != nil {
				return fmt.Errorf("failed to delete invalid group_items: %w", err)
			}
		}
		if hasPhysicalColumn(tx, "groups", "active_item_id") {
			if err := clearStaleActiveItems(tx); err != nil {
				return err
			}
		}

		// 旧唯一索引包含 channel_id 和 model_name, 删除列前先移除索引。
		if tx.Migrator().HasIndex(&model.GroupItem{}, "idx_group_channel_model") {
			if tx.Dialector.Name() == "mysql" && tx.Migrator().HasConstraint(&model.Group{}, "fk_groups_items") {
				if err := tx.Exec("ALTER TABLE group_items DROP FOREIGN KEY fk_groups_items").Error; err != nil {
					return fmt.Errorf("failed to drop group_items foreign key: %w", err)
				}
			}
			if err := tx.Migrator().DropIndex(&model.GroupItem{}, "idx_group_channel_model"); err != nil {
				return fmt.Errorf("failed to drop legacy group_items index: %w", err)
			}
		}
		for _, column := range []string{"channel_id", "model_name"} {
			if err := dropColumnIfExists(tx, &model.GroupItem{}, "group_items", column); err != nil {
				return err
			}
		}
		return nil
	})
}
