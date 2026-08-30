package op

import (
	"context"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

func TestDBImportIncrementalPreservesCacheTokenFieldsOnExistingChannel(t *testing.T) {
	ctx := context.Background()
	const channelID = 900001
	channel := model.Channel{ID: channelID, Name: "backup-stats-existing", AutoGroup: model.AutoGroupTypeNone}
	if err := db.GetDB().WithContext(ctx).Where("id = ?", channelID).Delete(&model.Channel{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Create(&channel).Error; err != nil {
		t.Fatal(err)
	}
	defer db.GetDB().WithContext(ctx).Where("id = ?", channelID).Delete(&model.Channel{})

	dump := &model.DBDump{
		Version: dbDumpVersion,
		Channels: []model.Channel{{
			ID:   channelID,
			Name: "backup-stats-existing",
			StatsMetrics: model.StatsMetrics{
				InputToken:      11,
				OutputToken:     13,
				CacheReadToken:  17,
				CacheWriteToken: 19,
			},
		}},
	}
	if _, err := DBImportIncremental(ctx, dump); err != nil {
		t.Fatal(err)
	}

	var got model.Channel
	if err := db.GetDB().WithContext(ctx).First(&got, channelID).Error; err != nil {
		t.Fatal(err)
	}
	if got.CacheReadToken != 17 || got.CacheWriteToken != 19 {
		t.Fatalf("cache token fields were not imported: read=%d write=%d", got.CacheReadToken, got.CacheWriteToken)
	}
}
