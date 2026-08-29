package op

import (
	"context"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

func TestStatsClearResetsPersistentAndMemoryStats(t *testing.T) {
	ctx := context.Background()
	if err := db.GetDB().WithContext(ctx).Where("1 = 1").Delete(&model.StatsDaily{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Where("1 = 1").Delete(&model.StatsHourly{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Where("1 = 1").Delete(&model.StatsAPIKey{}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.GetDB().WithContext(ctx).Save(&model.StatsTotal{
		ID:           1,
		StatsMetrics: model.StatsMetrics{InputToken: 10, CacheReadToken: 4, InputCost: 1.5},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Create(&model.StatsDaily{
		Date:         "20260828",
		StatsMetrics: model.StatsMetrics{RequestSuccess: 2},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Create(&model.StatsHourly{
		Hour:         1,
		Date:         "20260828",
		StatsMetrics: model.StatsMetrics{OutputToken: 3},
	}).Error; err != nil {
		t.Fatal(err)
	}

	statsTotalCacheLock.Lock()
	statsTotalCache = model.StatsTotal{ID: 1, StatsMetrics: model.StatsMetrics{InputToken: 9}}
	statsTotalCacheLock.Unlock()
	statsDailyCacheLock.Lock()
	statsDailyCache = model.StatsDaily{Date: "20260828", StatsMetrics: model.StatsMetrics{RequestFailed: 1}}
	statsDailyCacheLock.Unlock()
	statsHourlyCacheLock.Lock()
	statsHourlyCache[1] = model.StatsHourly{Hour: 1, Date: "20260828", StatsMetrics: model.StatsMetrics{OutputToken: 8}}
	statsHourlyCacheLock.Unlock()

	if err := StatsClear(ctx); err != nil {
		t.Fatal(err)
	}

	var total model.StatsTotal
	if err := db.GetDB().WithContext(ctx).First(&total, 1).Error; err != nil {
		t.Fatal(err)
	}
	if total.StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("persistent total not reset: %+v", total.StatsMetrics)
	}

	var dailyCount, hourlyCount, apiKeyCount int64
	if err := db.GetDB().WithContext(ctx).Model(&model.StatsDaily{}).Count(&dailyCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Model(&model.StatsHourly{}).Count(&hourlyCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Model(&model.StatsAPIKey{}).Count(&apiKeyCount).Error; err != nil {
		t.Fatal(err)
	}
	if dailyCount != 0 || hourlyCount != 0 || apiKeyCount != 0 {
		t.Fatalf("persistent detail stats not cleared: daily=%d hourly=%d apikey=%d", dailyCount, hourlyCount, apiKeyCount)
	}

	if got := StatsTotalGet().StatsMetrics; got != (model.StatsMetrics{}) {
		t.Fatalf("memory total not reset: %+v", got)
	}
	if got := StatsTodayGet().StatsMetrics; got != (model.StatsMetrics{}) {
		t.Fatalf("memory daily not reset: %+v", got)
	}
	if got := StatsHourlyGet(); len(got) == 0 || got[0].StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("memory hourly not reset: %+v", got)
	}
}
