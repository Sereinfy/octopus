package op

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	const channelID = 910001
	const channelModelID = 910002
	if err := db.GetDB().WithContext(ctx).Where("id = ?", channelID).Delete(&model.Channel{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Create(&model.Channel{ID: channelID, Name: "stats-clear-channel", AutoGroup: model.AutoGroupTypeNone, StatsMetrics: model.StatsMetrics{InputToken: 8}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GetDB().WithContext(ctx).Create(&model.ChannelModel{ID: channelModelID, ChannelID: channelID, Name: "stats-clear-model", StatsMetrics: model.StatsMetrics{OutputToken: 6}}).Error; err != nil {
		t.Fatal(err)
	}
	channelCache.Set(channelID, model.Channel{ID: channelID, Name: "stats-clear-channel", StatsMetrics: model.StatsMetrics{InputToken: 8}})
	channelModelCache.Set(channelModelID, model.ChannelModel{ID: channelModelID, ChannelID: channelID, Name: "stats-clear-model", StatsMetrics: model.StatsMetrics{OutputToken: 6}})
	defer func() {
		_ = db.GetDB().WithContext(ctx).Where("id = ?", channelID).Delete(&model.Channel{}).Error
		channelCache.Del(channelID)
		channelModelCache.Del(channelModelID)
	}()

	statsTotalCacheLock.Lock()
	statsTotalCache = model.StatsTotal{ID: 1, StatsMetrics: model.StatsMetrics{InputToken: 9}}
	statsTotalCacheLock.Unlock()
	statsDailyCacheLock.Lock()
	statsDailyCache = model.StatsDaily{Date: "20260828", StatsMetrics: model.StatsMetrics{RequestFailed: 1}}
	statsDailyCacheLock.Unlock()
	statsHourlyCacheLock.Lock()
	statsHourlyCache[1] = model.StatsHourly{Hour: 1, Date: "20260828", StatsMetrics: model.StatsMetrics{OutputToken: 8}}
	statsHourlyCacheLock.Unlock()
	statsAPIKeyCache.Set(910003, model.StatsAPIKey{APIKeyID: 910003, StatsMetrics: model.StatsMetrics{RequestSuccess: 4}})
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate[910003] = struct{}{}
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	defer func() {
		statsAPIKeyCache.Del(910003)
		statsAPIKeyCacheNeedUpdateLock.Lock()
		delete(statsAPIKeyCacheNeedUpdate, 910003)
		statsAPIKeyCacheNeedUpdateLock.Unlock()
	}()

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
	if got := StatsAPIKeyGet(910003).StatsMetrics; got != (model.StatsMetrics{}) {
		t.Fatalf("memory api key stats not reset: %+v", got)
	}
	if got := StatsHourlyGet(); len(got) == 0 || got[0].StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("memory hourly not reset: %+v", got)
	}
	if got, _ := ChannelGet(channelID); got.StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("memory channel not reset: %+v", got.StatsMetrics)
	}
	if got, _ := ChannelModelGet(channelModelID); got.StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("memory channel model not reset: %+v", got.StatsMetrics)
	}
	var channel model.Channel
	if err := db.GetDB().WithContext(ctx).First(&channel, channelID).Error; err != nil {
		t.Fatal(err)
	}
	if channel.StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("persistent channel not reset: %+v", channel.StatsMetrics)
	}
	var channelModel model.ChannelModel
	if err := db.GetDB().WithContext(ctx).First(&channelModel, channelModelID).Error; err != nil {
		t.Fatal(err)
	}
	if channelModel.StatsMetrics != (model.StatsMetrics{}) {
		t.Fatalf("persistent channel model not reset: %+v", channelModel.StatsMetrics)
	}
}

func TestStatsClearWaitsForInFlightStatsRecord(t *testing.T) {
	ctx := context.Background()
	if err := StatsClear(ctx); err != nil {
		t.Fatal(err)
	}
	statsLifecycleMu.RLock()
	if err := StatsRecord(ctx, model.StatsMetrics{RequestSuccess: 1}, nil, 0); err != nil {
		statsLifecycleMu.RUnlock()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- StatsClear(ctx) }()
	select {
	case err := <-done:
		statsLifecycleMu.RUnlock()
		t.Fatalf("clear interleaved with in-flight record, err=%v", err)
	case <-time.After(20 * time.Millisecond):
	}
	statsLifecycleMu.RUnlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := StatsTotalGet().StatsMetrics; got != (model.StatsMetrics{}) {
		t.Fatalf("clear did not reset record atomically: %+v", got)
	}
}

func TestStatsSummaryUsesCalendarRangeAndTodayCache(t *testing.T) {
	ctx := context.Background()
	today := time.Now()
	todayKey := today.Format("20060102")
	oldKey := today.AddDate(0, 0, -6).Format("20060102")
	statsDailyCacheLock.RLock()
	originalDailyCache := statsDailyCache
	statsDailyCacheLock.RUnlock()
	defer func() {
		statsDailyCacheLock.Lock()
		statsDailyCache = originalDailyCache
		statsDailyCacheLock.Unlock()
	}()
	for _, date := range []string{oldKey, todayKey} {
		if err := db.GetDB().WithContext(ctx).Where("date = ?", date).Delete(&model.StatsDaily{}).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer db.GetDB().WithContext(ctx).Where("date IN ?", []string{oldKey, todayKey}).Delete(&model.StatsDaily{})
	if err := db.GetDB().WithContext(ctx).Create(&model.StatsDaily{
		Date:         oldKey,
		StatsMetrics: model.StatsMetrics{RequestSuccess: 2, InputToken: 5},
	}).Error; err != nil {
		t.Fatal(err)
	}
	statsDailyCacheLock.Lock()
	statsDailyCache = model.StatsDaily{Date: todayKey, StatsMetrics: model.StatsMetrics{RequestFailed: 3, CacheReadToken: 7}}
	statsDailyCacheLock.Unlock()

	summary, err := StatsSummaryGet(ctx, "7")
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Points) != 7 {
		t.Fatalf("expected 7 calendar points, got %d", len(summary.Points))
	}
	if summary.Points[0].Date != today.AddDate(0, 0, -6).Format("20060102") {
		t.Fatalf("unexpected first point: %+v", summary.Points[0])
	}
	if got := summary.Points[0].RequestCount; got != 2 {
		t.Fatalf("expected persisted sparse day to be retained, got %d", got)
	}
	if got := summary.Points[6].RequestCount; got != 3 {
		t.Fatalf("expected today cache override, got %d", got)
	}
	if got := summary.StatsMetrics.CacheReadToken; got != 7 {
		t.Fatalf("expected today cache metric in summary, got %d", got)
	}
}

func TestStatsTodayGetRefreshesDateAfterMidnight(t *testing.T) {
	statsDailyCacheLock.RLock()
	originalDailyCache := statsDailyCache
	statsDailyCacheLock.RUnlock()
	defer func() {
		statsDailyCacheLock.Lock()
		statsDailyCache = originalDailyCache
		statsDailyCacheLock.Unlock()
	}()
	statsDailyCacheLock.Lock()
	statsDailyCache = model.StatsDaily{Date: "19990101", StatsMetrics: model.StatsMetrics{RequestSuccess: 9}}
	statsDailyCacheLock.Unlock()
	got := StatsTodayGet()
	if want := time.Now().Format("20060102"); got.Date != want || got.RequestSuccess != 0 {
		t.Fatalf("stale daily cache was not reset: got=%s success=%d want=%s", got.Date, got.RequestSuccess, want)
	}
	statsDailyCacheLock.RLock()
	deferredDate := statsDailyCache.Date
	deferredSuccess := statsDailyCache.RequestSuccess
	statsDailyCacheLock.RUnlock()
	if deferredDate != "19990101" || deferredSuccess != 9 {
		t.Fatalf("reading across midnight discarded prior-day cache: date=%s success=%d", deferredDate, deferredSuccess)
	}
}

func TestDownsampleSummaryPointsPreservesTotals(t *testing.T) {
	points := make([]model.StatsSummaryPoint, 0, 10)
	for i := 0; i < 10; i++ {
		points = append(points, model.StatsSummaryPoint{Date: fmt.Sprintf("202601%02d", i+1), TotalCost: 1, RequestCount: 2})
	}
	got := downsampleSummaryPoints(points, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(got))
	}
	var cost float64
	var requests int64
	for _, point := range got {
		cost += point.TotalCost
		requests += point.RequestCount
	}
	if cost != 10 || requests != 20 {
		t.Fatalf("downsample changed totals: cost=%v requests=%d", cost, requests)
	}
}
