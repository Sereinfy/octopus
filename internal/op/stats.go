package op

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var statsDailyCache model.StatsDaily
var statsDailyCacheLock sync.RWMutex

var statsTotalCache model.StatsTotal
var statsTotalCacheLock sync.RWMutex

var statsHourlyCache [24]model.StatsHourly
var statsHourlyCacheLock sync.RWMutex

var channelStatsNeedUpdate = make(map[int]struct{}) // 等待持久化的渠道 ID。
var channelStatsNeedUpdateLock sync.Mutex           // 保护渠道统计累加和待写集合。

var channelModelStatsNeedUpdate = make(map[int]struct{})
var channelModelStatsNeedUpdateLock sync.Mutex

var statsAPIKeyCache = cache.New[int, model.StatsAPIKey](16)
var statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
var statsAPIKeyCacheNeedUpdateLock sync.Mutex

// statsLifecycleMu serializes statistic updates with persistence and clearing.
// Read locks cover in-memory updates; write locks cover operations that take a
// snapshot or replace/reset multiple statistic stores as one lifecycle event.
var statsLifecycleMu sync.RWMutex

// ChannelStatsDelta groups one upstream round's channel and model metrics so
// the pair is committed under the same statistics lifecycle read lock.
type ChannelStatsDelta struct {
	ChannelID      int
	ChannelModelID int
	Metrics        model.StatsMetrics
}

func StatsSaveDBTask() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	log.Debugf("stats save db task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("stats save db task finished, save time: %s", time.Since(startTime))
	}()
	if err := StatsSaveDB(ctx); err != nil {
		log.Errorf("stats save db error: %v", err)
		return
	}
}

func StatsSaveDB(ctx context.Context) error {
	statsLifecycleMu.Lock()
	defer statsLifecycleMu.Unlock()

	statsTotalCacheLock.RLock()
	totalSnap := statsTotalCache
	statsTotalCacheLock.RUnlock()
	if totalSnap.ID == 0 {
		totalSnap.ID = 1
	}

	statsDailyCacheLock.RLock()
	dailySnap := statsDailyCache
	statsDailyCacheLock.RUnlock()

	statsHourlyCacheLock.RLock()
	hourlyAll := statsHourlyCache
	statsHourlyCacheLock.RUnlock()

	channelStatsNeedUpdateLock.Lock()
	channelIDs := make([]int, 0, len(channelStatsNeedUpdate))
	for id := range channelStatsNeedUpdate {
		channelIDs = append(channelIDs, id)
	}
	channelStatsNeedUpdate = make(map[int]struct{})
	channelStatsNeedUpdateLock.Unlock()

	channelModelStatsNeedUpdateLock.Lock()
	modelIDs := make([]int, 0, len(channelModelStatsNeedUpdate))
	for id := range channelModelStatsNeedUpdate {
		modelIDs = append(modelIDs, id)
	}
	channelModelStatsNeedUpdate = make(map[int]struct{})
	channelModelStatsNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	apiKeyIDs := make([]int, 0, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		apiKeyIDs = append(apiKeyIDs, id)
	}
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	if err := persistStatsSnapshots(ctx, totalSnap, dailySnap, hourlyAll, channelIDs, modelIDs, apiKeyIDs); err != nil {
		restoreStatsDirty(channelIDs, modelIDs, apiKeyIDs)
		return err
	}
	return nil
}

// restoreStatsDirty 在统计持久化失败后恢复本批待写标记。
func restoreStatsDirty(channelIDs, modelIDs, apiKeyIDs []int) {
	channelStatsNeedUpdateLock.Lock()
	for _, id := range channelIDs {
		channelStatsNeedUpdate[id] = struct{}{}
	}
	channelStatsNeedUpdateLock.Unlock()

	channelModelStatsNeedUpdateLock.Lock()
	for _, id := range modelIDs {
		channelModelStatsNeedUpdate[id] = struct{}{}
	}
	channelModelStatsNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	for _, id := range apiKeyIDs {
		statsAPIKeyCacheNeedUpdate[id] = struct{}{}
	}
	statsAPIKeyCacheNeedUpdateLock.Unlock()
}

func persistStatsSnapshots(
	ctx context.Context,
	totalSnap model.StatsTotal,
	dailySnap model.StatsDaily,
	hourlyAll [24]model.StatsHourly,
	channelIDs []int,
	modelIDs []int,
	apiKeyIDs []int,
) error {
	dbConn := db.GetDB().WithContext(ctx)

	if result := dbConn.Save(&totalSnap); result.Error != nil {
		return result.Error
	}
	if result := dbConn.Save(&dailySnap); result.Error != nil {
		return result.Error
	}

	todayDate := time.Now().Format("20060102")
	hourlyStats := make([]model.StatsHourly, 0, 24)
	for hour := 0; hour < 24; hour++ {
		if hourlyAll[hour].Date == todayDate {
			hourlyStats = append(hourlyStats, hourlyAll[hour])
		}
	}
	if len(hourlyStats) > 0 {
		if result := dbConn.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hour"}},
			UpdateAll: true,
		}).Create(&hourlyStats); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range channelIDs {
		channel, ok := channelCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Model(&model.Channel{}).
			Where("id = ?", channel.ID).
			Select("input_token", "output_token", "cache_read_token", "cache_write_token", "input_cost", "output_cost", "wait_time", "request_success", "request_failed").
			Updates(&channel); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range modelIDs {
		m, ok := channelModelCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Model(&model.ChannelModel{}).
			Where("id = ?", m.ID).
			Select("input_token", "output_token", "cache_read_token", "cache_write_token", "input_cost", "output_cost", "wait_time", "request_success", "request_failed").
			Updates(&m); result.Error != nil {
			return result.Error
		}
	}

	for _, id := range apiKeyIDs {
		ak, ok := statsAPIKeyCache.Get(id)
		if !ok {
			continue
		}
		if result := dbConn.Save(&ak); result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func statsSaveDBWithDailyOverride(ctx context.Context, dailyOverride model.StatsDaily) error {
	statsTotalCacheLock.RLock()
	totalSnap := statsTotalCache
	statsTotalCacheLock.RUnlock()
	if totalSnap.ID == 0 {
		totalSnap.ID = 1
	}

	statsHourlyCacheLock.RLock()
	hourlyAll := statsHourlyCache
	statsHourlyCacheLock.RUnlock()

	channelStatsNeedUpdateLock.Lock()
	channelIDs := make([]int, 0, len(channelStatsNeedUpdate))
	for id := range channelStatsNeedUpdate {
		channelIDs = append(channelIDs, id)
	}
	channelStatsNeedUpdate = make(map[int]struct{})
	channelStatsNeedUpdateLock.Unlock()

	channelModelStatsNeedUpdateLock.Lock()
	modelIDs := make([]int, 0, len(channelModelStatsNeedUpdate))
	for id := range channelModelStatsNeedUpdate {
		modelIDs = append(modelIDs, id)
	}
	channelModelStatsNeedUpdate = make(map[int]struct{})
	channelModelStatsNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	apiKeyIDs := make([]int, 0, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		apiKeyIDs = append(apiKeyIDs, id)
	}
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	if err := persistStatsSnapshots(ctx, totalSnap, dailyOverride, hourlyAll, channelIDs, modelIDs, apiKeyIDs); err != nil {
		restoreStatsDirty(channelIDs, modelIDs, apiKeyIDs)
		return err
	}
	return nil
}

func statsDailyUpdateUnlocked(ctx context.Context, metrics model.StatsMetrics) error {
	today := time.Now().Format("20060102")

	statsDailyCacheLock.Lock()
	if statsDailyCache.Date == today {
		statsDailyCache.StatsMetrics.Add(metrics)
		statsDailyCacheLock.Unlock()
		return nil
	}

	prevDaily := statsDailyCache
	statsDailyCache = model.StatsDaily{Date: today}
	statsDailyCache.StatsMetrics.Add(metrics)
	statsDailyCacheLock.Unlock()

	return statsSaveDBWithDailyOverride(ctx, prevDaily)
}

func statsTotalUpdateUnlocked(metrics model.StatsMetrics) {
	statsTotalCacheLock.Lock()
	defer statsTotalCacheLock.Unlock()
	if statsTotalCache.ID == 0 {
		statsTotalCache.ID = 1
	}
	statsTotalCache.StatsMetrics.Add(metrics)
}

func statsHourlyUpdateUnlocked(metrics model.StatsMetrics) {
	now := time.Now()
	nowHour := now.Hour()
	todayDate := time.Now().Format("20060102")

	statsHourlyCacheLock.Lock()
	defer statsHourlyCacheLock.Unlock()

	if statsHourlyCache[nowHour].Date != todayDate {
		statsHourlyCache[nowHour] = model.StatsHourly{
			Hour: nowHour,
			Date: todayDate,
		}
	}

	statsHourlyCache[nowHour].StatsMetrics.Add(metrics)
}

func channelStatsUpdateUnlocked(channelID int, metrics model.StatsMetrics) {
	channelStatsNeedUpdateLock.Lock()
	defer channelStatsNeedUpdateLock.Unlock()
	channel, ok := channelCache.Get(channelID)
	if !ok {
		return
	}
	channel.StatsMetrics.Add(metrics)
	channelCache.Set(channelID, channel)
	channelStatsNeedUpdate[channelID] = struct{}{}
}

func channelModelStatsUpdateUnlocked(channelModelID int, metrics model.StatsMetrics) {
	channelModelStatsNeedUpdateLock.Lock()
	defer channelModelStatsNeedUpdateLock.Unlock()
	channelModel, ok := channelModelCache.Get(channelModelID)
	if !ok {
		return
	}
	channelModel.StatsMetrics.Add(metrics)
	channelModelCache.Set(channelModelID, channelModel)
	channelModelStatsNeedUpdate[channelModelID] = struct{}{}
}

func statsAPIKeyUpdateUnlocked(apiKeyID int, metrics model.StatsMetrics) {
	statsAPIKeyCacheNeedUpdateLock.Lock()
	defer statsAPIKeyCacheNeedUpdateLock.Unlock()
	apiKeyCache, ok := statsAPIKeyCache.Get(apiKeyID)
	if !ok {
		apiKeyCache = model.StatsAPIKey{
			APIKeyID: apiKeyID,
		}
	}
	apiKeyCache.StatsMetrics.Add(metrics)
	statsAPIKeyCache.Set(apiKeyID, apiKeyCache)
	statsAPIKeyCacheNeedUpdate[apiKeyID] = struct{}{}
}

// StatsDailyUpdate 累加每日统计并在跨日时持久化前一天快照。
func StatsDailyUpdate(ctx context.Context, metrics model.StatsMetrics) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsDailyUpdateUnlocked(ctx, metrics)
}

func StatsTotalUpdate(metrics model.StatsMetrics) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	statsTotalUpdateUnlocked(metrics)
	return nil
}

func StatsHourlyUpdate(metrics model.StatsMetrics) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	statsHourlyUpdateUnlocked(metrics)
	return nil
}

// ChannelStatsUpdate 累加渠道统计并标记对应渠道待持久化。
func ChannelStatsUpdate(channelID int, metrics model.StatsMetrics) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	channelStatsUpdateUnlocked(channelID, metrics)
	return nil
}

// ChannelModelStatsUpdate 累加渠道模型统计并标记对应模型待持久化。
func ChannelModelStatsUpdate(channelModelID int, metrics model.StatsMetrics) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	channelModelStatsUpdateUnlocked(channelModelID, metrics)
	return nil
}

func StatsAPIKeyUpdate(apiKeyID int, metrics model.StatsMetrics) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	statsAPIKeyUpdateUnlocked(apiKeyID, metrics)
	return nil
}

// StatsRecord atomically records one logical request and all of its upstream
// round deltas relative to a concurrent statistics clear.
func StatsRecord(ctx context.Context, metrics model.StatsMetrics, channelDeltas []ChannelStatsDelta, apiKeyID int) error {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()

	statsTotalUpdateUnlocked(metrics)
	statsHourlyUpdateUnlocked(metrics)
	dailyErr := statsDailyUpdateUnlocked(ctx, metrics)
	for _, delta := range channelDeltas {
		channelStatsUpdateUnlocked(delta.ChannelID, delta.Metrics)
		channelModelStatsUpdateUnlocked(delta.ChannelModelID, delta.Metrics)
	}
	if apiKeyID > 0 {
		statsAPIKeyUpdateUnlocked(apiKeyID, metrics)
	}
	return dailyErr
}

func StatsAPIKeyDel(id int) error {
	statsLifecycleMu.Lock()
	defer statsLifecycleMu.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	if _, ok := statsAPIKeyCache.Get(id); !ok {
		statsAPIKeyCacheNeedUpdateLock.Unlock()
		return nil
	}
	statsAPIKeyCache.Del(id)
	delete(statsAPIKeyCacheNeedUpdate, id)
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	return db.GetDB().Delete(&model.StatsAPIKey{}, id).Error
}

// StatsClear 清空所有统计数据，但保留渠道、模型和 API Key 配置。
func StatsClear(ctx context.Context) error {
	statsLifecycleMu.Lock()
	defer statsLifecycleMu.Unlock()

	resetValues := map[string]any{
		"input_token":       int64(0),
		"output_token":      int64(0),
		"cache_read_token":  int64(0),
		"cache_write_token": int64(0),
		"input_cost":        float64(0),
		"output_cost":       float64(0),
		"wait_time":         int64(0),
		"request_success":   int64(0),
		"request_failed":    int64(0),
	}

	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		clear := tx.Session(&gorm.Session{AllowGlobalUpdate: true})
		if err := clear.Delete(&model.StatsDaily{}).Error; err != nil {
			return err
		}
		if err := clear.Delete(&model.StatsHourly{}).Error; err != nil {
			return err
		}
		if err := clear.Delete(&model.StatsAPIKey{}).Error; err != nil {
			return err
		}
		if err := clear.Model(&model.StatsTotal{}).Where("id = ?", 1).Updates(resetValues).Error; err != nil {
			return err
		}
		if err := clear.Model(&model.Channel{}).Updates(resetValues).Error; err != nil {
			return err
		}
		return clear.Model(&model.ChannelModel{}).Updates(resetValues).Error
	}); err != nil {
		return err
	}

	today := time.Now().Format("20060102")
	statsTotalCacheLock.Lock()
	statsTotalCache = model.StatsTotal{ID: 1}
	statsTotalCacheLock.Unlock()

	statsDailyCacheLock.Lock()
	statsDailyCache = model.StatsDaily{Date: today}
	statsDailyCacheLock.Unlock()

	statsHourlyCacheLock.Lock()
	statsHourlyCache = [24]model.StatsHourly{}
	statsHourlyCacheLock.Unlock()

	statsAPIKeyCache.Clear()
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	channelStatsNeedUpdateLock.Lock()
	for id, channel := range channelCache.GetAll() {
		channel.StatsMetrics = model.StatsMetrics{}
		channelCache.Set(id, channel)
	}
	channelStatsNeedUpdate = make(map[int]struct{})
	channelStatsNeedUpdateLock.Unlock()

	channelModelStatsNeedUpdateLock.Lock()
	for id, channelModel := range channelModelCache.GetAll() {
		channelModel.StatsMetrics = model.StatsMetrics{}
		channelModelCache.Set(id, channelModel)
	}
	channelModelStatsNeedUpdate = make(map[int]struct{})
	channelModelStatsNeedUpdateLock.Unlock()

	return nil
}

func statsTotalGetUnlocked() model.StatsTotal {
	statsTotalCacheLock.RLock()
	defer statsTotalCacheLock.RUnlock()
	return statsTotalCache
}

func StatsTotalGet() model.StatsTotal {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsTotalGetUnlocked()
}

func statsTodayGetUnlocked() model.StatsDaily {
	today := time.Now().Format("20060102")
	statsDailyCacheLock.RLock()
	defer statsDailyCacheLock.RUnlock()
	if statsDailyCache.Date != today {
		return model.StatsDaily{Date: today}
	}
	return statsDailyCache
}

func StatsTodayGet() model.StatsDaily {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsTodayGetUnlocked()
}

func statsAPIKeyGetUnlocked(id int) model.StatsAPIKey {
	if stats, ok := statsAPIKeyCache.Get(id); ok {
		return stats
	}
	statsAPIKeyCacheNeedUpdateLock.Lock()
	defer statsAPIKeyCacheNeedUpdateLock.Unlock()
	stats, ok := statsAPIKeyCache.Get(id)
	if !ok {
		tmp := model.StatsAPIKey{
			APIKeyID: id,
		}
		statsAPIKeyCache.Set(id, tmp)
		statsAPIKeyCacheNeedUpdate[id] = struct{}{}
		return tmp
	}
	return stats
}

func StatsAPIKeyGet(id int) model.StatsAPIKey {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsAPIKeyGetUnlocked(id)
}

func statsAPIKeyListUnlocked() []model.StatsAPIKey {
	apiKeys := make([]model.StatsAPIKey, 0, statsAPIKeyCache.Len())
	for _, v := range statsAPIKeyCache.GetAll() {
		apiKeys = append(apiKeys, v)
	}
	return apiKeys
}

func StatsAPIKeyList() []model.StatsAPIKey {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsAPIKeyListUnlocked()
}

func statsHourlyGetUnlocked() []model.StatsHourly {
	now := time.Now()
	currentHour := now.Hour()
	todayDate := time.Now().Format("20060102")

	statsHourlyCacheLock.RLock()
	defer statsHourlyCacheLock.RUnlock()

	result := make([]model.StatsHourly, 0, currentHour+1)

	for hour := 0; hour <= currentHour; hour++ {
		if statsHourlyCache[hour].Date == todayDate {
			result = append(result, statsHourlyCache[hour])
		} else {
			result = append(result, model.StatsHourly{
				Hour: hour,
				Date: todayDate,
			})
		}
	}

	return result
}

func StatsHourlyGet() []model.StatsHourly {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsHourlyGetUnlocked()
}

func StatsGetDaily(ctx context.Context) ([]model.StatsDaily, error) {
	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()
	return statsGetDailyUnlocked(ctx)
}

func statsGetDailyUnlocked(ctx context.Context) ([]model.StatsDaily, error) {
	var statsDaily []model.StatsDaily
	result := db.GetDB().WithContext(ctx).Order("date asc").Find(&statsDaily)
	if result.Error != nil {
		return nil, result.Error
	}
	return statsDaily, nil
}

// StatsSummaryGet 聚合首页汇总所需的数据，并将今日内存统计覆盖到每日快照。
func StatsSummaryGet(ctx context.Context, period string) (model.StatsSummary, error) {
	if period != "1" && period != "7" && period != "30" && period != "all" {
		return model.StatsSummary{}, fmt.Errorf("invalid stats summary period: %s", period)
	}

	statsLifecycleMu.RLock()
	defer statsLifecycleMu.RUnlock()

	if period == "1" {
		hourly := statsHourlyGetUnlocked()
		var metrics model.StatsMetrics
		points := make([]model.StatsSummaryPoint, 0, len(hourly))
		for _, stat := range hourly {
			metrics.Add(stat.StatsMetrics)
			points = append(points, model.StatsSummaryPoint{
				Date:         fmt.Sprintf("%d:00", stat.Hour),
				TotalCost:    stat.InputCost + stat.OutputCost,
				RequestCount: stat.RequestSuccess + stat.RequestFailed,
			})
		}
		return model.StatsSummary{Period: period, StatsMetrics: metrics, Points: points}, nil
	}

	today := statsTodayGetUnlocked()
	var daily []model.StatsDaily
	var err error
	if period == "all" {
		daily, err = statsGetDailyUnlocked(ctx)
	} else {
		days := 7
		if period == "30" {
			days = 30
		}
		todayDate := time.Now()
		startDate := todayDate.AddDate(0, 0, -(days - 1)).Format("20060102")
		endDate := todayDate.Format("20060102")
		err = db.GetDB().WithContext(ctx).
			Where("date BETWEEN ? AND ?", startDate, endDate).
			Order("date asc").Find(&daily).Error
		if err == nil {
			byDate := make(map[string]model.StatsDaily, len(daily)+1)
			for _, stat := range daily {
				byDate[stat.Date] = stat
			}
			filled := make([]model.StatsDaily, 0, days)
			for date := todayDate.AddDate(0, 0, -(days - 1)); !date.After(todayDate); date = date.AddDate(0, 0, 1) {
				key := date.Format("20060102")
				stat, ok := byDate[key]
				if !ok {
					stat = model.StatsDaily{Date: key}
				}
				if key == today.Date {
					stat = today
				}
				filled = append(filled, stat)
			}
			daily = filled
		}
	}
	if err != nil {
		return model.StatsSummary{}, err
	}
	if period == "all" {
		foundToday := false
		for i := range daily {
			if daily[i].Date == today.Date {
				daily[i] = today
				foundToday = true
				break
			}
		}
		if !foundToday {
			daily = append(daily, today)
		}
		sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })
	}

	var metrics model.StatsMetrics
	points := make([]model.StatsSummaryPoint, 0, len(daily))
	for _, stat := range daily {
		metrics.Add(stat.StatsMetrics)
		points = append(points, model.StatsSummaryPoint{
			Date:         stat.Date,
			TotalCost:    stat.InputCost + stat.OutputCost,
			RequestCount: stat.RequestSuccess + stat.RequestFailed,
		})
	}
	if period == "all" {
		points = downsampleSummaryPoints(points, 366)
	}
	if period == "all" {
		metrics = statsTotalGetUnlocked().StatsMetrics
	}
	return model.StatsSummary{Period: period, StatsMetrics: metrics, Points: points}, nil
}

// downsampleSummaryPoints bounds the all-time chart payload while preserving
// the total cost and request count represented by every bucket.
func downsampleSummaryPoints(points []model.StatsSummaryPoint, maxPoints int) []model.StatsSummaryPoint {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	bucketSize := (len(points) + maxPoints - 1) / maxPoints
	result := make([]model.StatsSummaryPoint, 0, (len(points)+bucketSize-1)/bucketSize)
	for start := 0; start < len(points); start += bucketSize {
		end := start + bucketSize
		if end > len(points) {
			end = len(points)
		}
		bucket := model.StatsSummaryPoint{Date: points[start].Date}
		for _, point := range points[start:end] {
			bucket.TotalCost += point.TotalCost
			bucket.RequestCount += point.RequestCount
		}
		result = append(result, bucket)
	}
	return result
}

func statsRefreshCache(ctx context.Context) error {
	dbConn := db.GetDB().WithContext(ctx)
	today := time.Now().Format("20060102")

	var loadedDaily model.StatsDaily
	result := dbConn.Last(&loadedDaily)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get daily stats: %v", result.Error)
	}
	if result.RowsAffected == 0 || loadedDaily.Date != today {
		loadedDaily = model.StatsDaily{Date: today}
	}

	var loadedTotal model.StatsTotal
	result = dbConn.First(&loadedTotal)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get total stats: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		loadedTotal = model.StatsTotal{ID: 1}
	} else if loadedTotal.ID == 0 {
		loadedTotal.ID = 1
	}

	var loadedHourly []model.StatsHourly
	result = dbConn.Find(&loadedHourly)
	if result.Error != nil {
		return fmt.Errorf("failed to get hourly stats: %v", result.Error)
	}

	statsDailyCacheLock.Lock()
	statsDailyCache = loadedDaily
	statsDailyCacheLock.Unlock()

	statsTotalCacheLock.Lock()
	statsTotalCache = loadedTotal
	statsTotalCacheLock.Unlock()

	var loadedAPIKeys []model.StatsAPIKey
	result = dbConn.Find(&loadedAPIKeys)
	if result.Error != nil {
		return fmt.Errorf("failed to get api key stats: %v", result.Error)
	}

	statsAPIKeyCache.Clear()
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate = make(map[int]struct{})
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	for _, v := range loadedAPIKeys {
		statsAPIKeyCache.Set(v.APIKeyID, v)
	}

	statsHourlyCacheLock.Lock()
	statsHourlyCache = [24]model.StatsHourly{}
	for _, v := range loadedHourly {
		if v.Hour >= 0 && v.Hour < 24 {
			statsHourlyCache[v.Hour] = v
		}
	}
	statsHourlyCacheLock.Unlock()

	return nil
}
