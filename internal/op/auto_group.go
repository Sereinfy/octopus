package op

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/dlclark/regexp2"
	"gorm.io/gorm"
)

var autoGroupMu sync.Mutex

func validateGroupMatchRegex(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}
	if _, err := regexp2.Compile(pattern, regexp2.ECMAScript); err != nil {
		return fmt.Errorf("invalid auto group regex: %w", err)
	}
	return nil
}

type AutoGroupResult struct {
	Channels int `json:"channels"`
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Created  int `json:"created"`
}

func AutoGroupGlobalMode() model.AutoGroupType {
	value, err := SettingGetString(model.SettingKeyAutoGroupGlobalMode)
	if err != nil {
		return model.AutoGroupTypeNone
	}
	mode, ok := model.ParseAutoGroupSettingValue(value)
	if !ok {
		return model.AutoGroupTypeNone
	}
	return mode
}

func AutoGroupCreateMissingEnabled() bool {
	enabled, err := SettingGetBool(model.SettingKeyAutoGroupCreateMissingEnabled)
	return err == nil && enabled
}

func AutoGroupNormalizeEnabled() bool {
	enabled, err := SettingGetBool(model.SettingKeyAutoGroupNormalizeEnabled)
	return err == nil && enabled
}

func GroupAutoGroupConfigGet(ctx context.Context) (*model.GroupAutoGroupConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	channels := ChannelList()
	sources := make([]model.GroupAutoGroupSource, 0, len(channels))
	for _, channel := range channels {
		models := make([]string, 0, len(channel.Models))
		for _, channelModel := range channel.Models {
			models = append(models, channelModel.Name)
		}
		sources = append(sources, model.GroupAutoGroupSource{
			ChannelID: channel.ID, ChannelName: channel.Name, Enabled: channel.Enabled,
			AutoGroup: channel.AutoGroup, ModelCount: len(models), Models: models,
		})
	}
	sort.Slice(sources, func(i, j int) bool {
		if strings.EqualFold(sources[i].ChannelName, sources[j].ChannelName) {
			return sources[i].ChannelID < sources[j].ChannelID
		}
		return strings.ToLower(sources[i].ChannelName) < strings.ToLower(sources[j].ChannelName)
	})
	return &model.GroupAutoGroupConfig{
		GlobalMode:          AutoGroupGlobalMode(),
		CreateMissingGroups: AutoGroupCreateMissingEnabled(),
		NormalizeModelNames: AutoGroupNormalizeEnabled(),
		Sources:             sources,
	}, nil
}

func GroupAutoGroupConfigUpdate(req *model.GroupAutoGroupConfigUpdateRequest, ctx context.Context) (*model.GroupAutoGroupConfig, error) {
	if req == nil {
		return nil, fmt.Errorf("auto group config request is required")
	}
	seen := make(map[int]struct{}, len(req.Items))
	for _, item := range req.Items {
		if item.ChannelID <= 0 {
			return nil, fmt.Errorf("channel id is required")
		}
		if _, exists := seen[item.ChannelID]; exists {
			return nil, fmt.Errorf("duplicate channel: %d", item.ChannelID)
		}
		seen[item.ChannelID] = struct{}{}
		if _, err := ChannelGet(item.ChannelID); err != nil {
			return nil, err
		}
		if item.AutoGroup != nil && !item.AutoGroup.Valid() {
			return nil, fmt.Errorf("invalid auto group type")
		}
	}
	if req.GlobalMode != nil && !req.GlobalMode.Valid() {
		return nil, fmt.Errorf("invalid global auto group type")
	}
	if req.GlobalMode != nil {
		if err := SettingSetString(model.SettingKeyAutoGroupGlobalMode, fmt.Sprintf("%d", *req.GlobalMode)); err != nil {
			return nil, err
		}
		// A global-only update is also a convenient batch operation for API clients.
		if len(req.Items) == 0 {
			for _, channel := range ChannelList() {
				if err := ChannelAutoGroupUpdate(channel.ID, *req.GlobalMode, ctx); err != nil {
					return nil, err
				}
			}
		}
	}
	if req.CreateMissingGroups != nil {
		value := "false"
		if *req.CreateMissingGroups {
			value = "true"
		}
		if err := SettingSetString(model.SettingKeyAutoGroupCreateMissingEnabled, value); err != nil {
			return nil, err
		}
	}
	if req.NormalizeModelNames != nil {
		value := "false"
		if *req.NormalizeModelNames {
			value = "true"
		}
		if err := SettingSetString(model.SettingKeyAutoGroupNormalizeEnabled, value); err != nil {
			return nil, err
		}
	}
	for _, item := range req.Items {
		if item.AutoGroup == nil {
			continue
		}
		if err := ChannelAutoGroupUpdate(item.ChannelID, *item.AutoGroup, ctx); err != nil {
			return nil, err
		}
	}
	if req.RunNow {
		if _, err := RunGroupAutoGroup(nil, ctx); err != nil {
			return nil, err
		}
	}
	return GroupAutoGroupConfigGet(ctx)
}

func ChannelAutoGroupUpdate(channelID int, mode model.AutoGroupType, ctx context.Context) error {
	if !mode.Valid() {
		return fmt.Errorf("invalid auto group type")
	}
	if _, err := ChannelGet(channelID); err != nil {
		return err
	}
	if err := db.GetDB().WithContext(ctx).Model(&model.Channel{}).Where("id = ?", channelID).Update("auto_group", mode).Error; err != nil {
		return err
	}
	return channelRefreshCacheByID(channelID, ctx)
}

func RunGroupAutoGroup(channelIDs []int, ctx context.Context) (*AutoGroupResult, error) {
	autoGroupMu.Lock()
	defer autoGroupMu.Unlock()
	all := channelCache.GetAll()
	targets := make([]model.Channel, 0, len(all))
	if len(channelIDs) == 0 {
		for _, channel := range all {
			targets = append(targets, channelSnapshot(channel))
		}
	} else {
		seen := make(map[int]struct{}, len(channelIDs))
		for _, id := range channelIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			channel, ok := all[id]
			if !ok {
				return nil, fmt.Errorf("channel not found: %d", id)
			}
			targets = append(targets, channelSnapshot(channel))
		}
	}
	result := &AutoGroupResult{}
	for i := range targets {
		mode := targets[i].AutoGroup
		if mode == model.AutoGroupTypeNone {
			continue
		}
		added, removed, created, err := autoGroupChannel(&targets[i], mode, ctx)
		if err != nil {
			return result, err
		}
		result.Channels++
		result.Added += added
		result.Removed += removed
		result.Created += created
	}
	return result, nil
}

func autoGroupChannel(channel *model.Channel, mode model.AutoGroupType, ctx context.Context) (int, int, int, error) {
	if channel == nil || mode == model.AutoGroupTypeNone {
		return 0, 0, 0, nil
	}
	groups := GroupList()
	added, removed := 0, 0
	for _, group := range groups {
		desired, err := matchingModelIDs(group, channel.Models, mode)
		if err != nil {
			return added, removed, 0, err
		}
		a, r, err := reconcileAutoItems(group, channel.ID, desired, ctx)
		if err != nil {
			return added, removed, 0, err
		}
		added += a
		removed += r
	}
	created := 0
	if mode == model.AutoGroupTypeExact && AutoGroupCreateMissingEnabled() {
		createdGroups, err := createMissingGroups(channel, groups, ctx)
		if err != nil {
			return added, removed, created, err
		}
		created = createdGroups
	}
	return added, removed, created, nil
}

func matchingModelIDs(group model.Group, models []model.ChannelModel, mode model.AutoGroupType) ([]int, error) {
	matched := make([]int, 0)
	groupName := strings.TrimSpace(group.Name)
	if groupName == "" {
		return matched, nil
	}
	var re *regexp2.Regexp
	if mode == model.AutoGroupTypeRegex && strings.TrimSpace(group.MatchRegex) != "" {
		compiled, err := regexp2.Compile(group.MatchRegex, regexp2.ECMAScript)
		if err != nil {
			return nil, fmt.Errorf("group %d regex: %w", group.ID, err)
		}
		compiled.MatchTimeout = 200 * time.Millisecond
		re = compiled
	}
	for _, channelModel := range models {
		name := strings.TrimSpace(channelModel.Name)
		if name == "" {
			continue
		}
		match := false
		switch mode {
		case model.AutoGroupTypeExact:
			match = publicModelNamesMatch(name, groupName)
		case model.AutoGroupTypeFuzzy:
			match = strings.Contains(strings.ToLower(name), strings.ToLower(groupName))
		case model.AutoGroupTypeRegex:
			if re == nil {
				match = publicModelNamesMatch(name, groupName)
			} else {
				var err error
				match, err = re.MatchString(name)
				if err != nil {
					return nil, err
				}
			}
		}
		if match {
			matched = append(matched, channelModel.ID)
		}
	}
	return matched, nil
}

func reconcileAutoItems(group model.Group, channelID int, desired []int, ctx context.Context) (int, int, error) {
	desiredSet := make(map[int]struct{}, len(desired))
	for _, id := range desired {
		desiredSet[id] = struct{}{}
	}
	var addIDs []int
	var removeIDs []int
	for _, item := range group.Items {
		if item.ChannelModel == nil || item.ChannelModel.ChannelID != channelID {
			continue
		}
		if _, ok := desiredSet[item.ChannelModelID]; ok {
			continue
		}
		if item.Source == model.GroupItemSourceAuto {
			removeIDs = append(removeIDs, item.ID)
		}
	}
	for _, id := range desired {
		found := false
		for _, item := range group.Items {
			if item.ChannelModelID == id {
				found = true
				break
			}
		}
		if !found {
			addIDs = append(addIDs, id)
		}
	}
	if len(removeIDs) == 0 && len(addIDs) == 0 {
		return 0, 0, nil
	}
	actualAdded := int64(0)
	actualRemoved := int64(0)
	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(removeIDs) > 0 {
			if err := tx.Model(&model.Group{}).
				Where("id = ? AND active_item_id IN ?", group.ID, removeIDs).
				Update("active_item_id", 0).Error; err != nil {
				return err
			}
			result := tx.Where("id IN ? AND source = ?", removeIDs, model.GroupItemSourceAuto).Delete(&model.GroupItem{})
			if result.Error != nil {
				return result.Error
			}
			actualRemoved = result.RowsAffected
		}
		if len(addIDs) > 0 {
			maxPriority := 0
			for _, item := range group.Items {
				if item.Priority > maxPriority {
					maxPriority = item.Priority
				}
			}
			items := make([]model.GroupItem, 0, len(addIDs))
			for i, id := range addIDs {
				items = append(items, model.GroupItem{GroupID: group.ID, ChannelModelID: id, Priority: maxPriority + i + 1, Source: model.GroupItemSourceAuto})
			}
			result := tx.Create(&items)
			if result.Error != nil {
				return result.Error
			}
			actualAdded = result.RowsAffected
		}
		return nil
	}); err != nil {
		return 0, 0, err
	}
	if err := groupRefreshCache(ctx); err != nil {
		return 0, 0, err
	}
	return int(actualAdded), int(actualRemoved), nil
}

func createMissingGroups(channel *model.Channel, existing []model.Group, ctx context.Context) (int, error) {
	existingNames := make(map[string]struct{}, len(existing))
	for _, group := range existing {
		existingNames[strings.ToLower(strings.TrimSpace(group.Name))] = struct{}{}
	}
	pending := make(map[string]struct {
		name string
		ids  []int
	})
	for _, channelModel := range channel.Models {
		name := strings.TrimSpace(channelModel.Name)
		if name == "" {
			continue
		}
		groupName := normalizePublicModelName(name)
		if groupName == "" {
			groupName = name
		}
		key := strings.ToLower(groupName)
		if _, ok := existingNames[key]; ok {
			continue
		}
		entry := pending[key]
		entry.name = groupName
		entry.ids = append(entry.ids, channelModel.ID)
		pending[key] = entry
	}
	created := 0
	for key, entry := range pending {
		err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			group := model.Group{Name: entry.name, Mode: model.GroupModeManual, RelayConfig: model.DefaultGroupRelayConfig()}
			if err := tx.Create(&group).Error; err != nil {
				return err
			}
			items := make([]model.GroupItem, 0, len(entry.ids))
			for i, id := range entry.ids {
				items = append(items, model.GroupItem{GroupID: group.ID, ChannelModelID: id, Priority: i + 1, Source: model.GroupItemSourceAuto})
			}
			return tx.Create(&items).Error
		})
		if err != nil {
			return created, err
		}
		existingNames[key] = struct{}{}
		created++
	}
	if created > 0 {
		if err := groupRefreshCache(ctx); err != nil {
			return created, err
		}
	}
	return created, nil
}

var dateSuffix = regexp.MustCompile(`(?i)-\d{4}-\d{2}-\d{2}$`)

func normalizePublicModelName(name string) string {
	value := strings.TrimSpace(name)
	if i := strings.IndexByte(value, '/'); i >= 0 {
		value = value[i+1:]
	}
	value = dateSuffix.ReplaceAllString(value, "")
	return strings.TrimSpace(value)
}

func publicModelNamesMatch(modelName, groupName string) bool {
	if strings.EqualFold(strings.TrimSpace(modelName), strings.TrimSpace(groupName)) {
		return true
	}
	if !AutoGroupNormalizeEnabled() {
		return false
	}
	return strings.EqualFold(normalizePublicModelName(modelName), normalizePublicModelName(groupName))
}
