package op

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "octopus-auto-group-test-")
	if err != nil {
		panic(err)
	}
	if err := db.InitDB("sqlite", filepath.Join(dir, "test.db"), false); err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err := settingRefreshCache(ctx); err != nil {
		panic(err)
	}
	if err := channelRefreshCache(ctx); err != nil {
		panic(err)
	}
	if err := groupRefreshCache(ctx); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = db.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func resetAutoGroupTestState(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{"group_items", "groups", "channel_models", "channels"} {
		if err := db.GetDB().WithContext(ctx).Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	for key, value := range map[model.SettingKey]string{
		model.SettingKeyAutoGroupGlobalMode:           "0",
		model.SettingKeyAutoGroupCreateMissingEnabled: "false",
		model.SettingKeyAutoGroupNormalizeEnabled:     "false",
	} {
		if err := SettingSetString(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := channelRefreshCache(ctx); err != nil {
		t.Fatal(err)
	}
	if err := groupRefreshCache(ctx); err != nil {
		t.Fatal(err)
	}
}

func createAutoGroupTestChannel(t *testing.T, name string, mode model.AutoGroupType, modelNames ...string) *model.Channel {
	t.Helper()
	channel := &model.Channel{Name: name, AutoGroup: mode}
	for _, modelName := range modelNames {
		channel.Models = append(channel.Models, model.ChannelModel{Name: modelName, Source: model.ChannelModelSourceManual})
	}
	if err := ChannelCreate(channel, context.Background()); err != nil {
		t.Fatal(err)
	}
	return channel
}

func TestMatchingModelIDs(t *testing.T) {
	resetAutoGroupTestState(t)
	models := []model.ChannelModel{
		{ID: 1, Name: "gpt-4"},
		{ID: 2, Name: "GPT-4-Turbo"},
		{ID: 3, Name: "claude-3"},
	}

	tests := []struct {
		name  string
		group model.Group
		mode  model.AutoGroupType
		want  []int
	}{
		{name: "exact", group: model.Group{Name: "GPT-4"}, mode: model.AutoGroupTypeExact, want: []int{1}},
		{name: "fuzzy", group: model.Group{Name: "gpt-4"}, mode: model.AutoGroupTypeFuzzy, want: []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchingModelIDs(tt.group, models, tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunGroupAutoGroupProtectsManualItemsAndClearsStaleActiveItem(t *testing.T) {
	resetAutoGroupTestState(t)
	channel := createAutoGroupTestChannel(t, "primary", model.AutoGroupTypeExact, "gpt-4", "stale-auto", "manual-only")
	group := &model.Group{
		Name: "gpt-4",
		Mode: model.GroupModeManual,
		Items: []model.GroupItem{
			{ChannelModelID: channel.Models[1].ID, Priority: 1, Source: model.GroupItemSourceAuto},
			{ChannelModelID: channel.Models[2].ID, Priority: 2, Source: model.GroupItemSourceManual},
		},
	}
	if err := GroupCreate(group, context.Background()); err != nil {
		t.Fatal(err)
	}
	staleItemID := group.Items[0].ID
	if _, err := GroupActiveItemUpdate(group.ID, &model.GroupActiveItemUpdateRequest{ItemID: &staleItemID}, context.Background()); err != nil {
		t.Fatal(err)
	}

	result, err := RunGroupAutoGroup([]int{channel.ID}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || result.Removed != 1 {
		t.Fatalf("unexpected result: %+v groups=%+v channel=%+v", result, GroupList(), channel)
	}

	groups := GroupList()
	if len(groups) != 1 {
		t.Fatalf("got %d groups", len(groups))
	}
	if groups[0].ActiveItemID != 0 {
		t.Fatalf("stale active item was not cleared: %d", groups[0].ActiveItemID)
	}
	items := make(map[int]model.GroupItemSource)
	for _, item := range groups[0].Items {
		items[item.ChannelModelID] = item.Source
	}
	if items[channel.Models[0].ID] != model.GroupItemSourceAuto {
		t.Fatalf("matched model was not added as auto: %v", items)
	}
	if items[channel.Models[2].ID] != model.GroupItemSourceManual {
		t.Fatalf("manual item was removed or changed: %v", items)
	}
	if _, exists := items[channel.Models[1].ID]; exists {
		t.Fatalf("stale auto item was not removed: %v", items)
	}
}

func TestRunGroupAutoGroupCreatesOneNormalizedGroupWithAllModels(t *testing.T) {
	resetAutoGroupTestState(t)
	if err := SettingSetString(model.SettingKeyAutoGroupCreateMissingEnabled, "true"); err != nil {
		t.Fatal(err)
	}
	if err := SettingSetString(model.SettingKeyAutoGroupNormalizeEnabled, "true"); err != nil {
		t.Fatal(err)
	}
	channel := createAutoGroupTestChannel(t, "normalized", model.AutoGroupTypeExact,
		"openai/gpt-4-2025-01-01", "gpt-4")

	result, err := RunGroupAutoGroup([]int{channel.ID}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 1 {
		t.Fatalf("unexpected result: %+v groups=%+v channel=%+v", result, GroupList(), channel)
	}
	groups := GroupList()
	if len(groups) != 1 || groups[0].Name != "gpt-4" || len(groups[0].Items) != 2 {
		t.Fatalf("unexpected groups: %+v", groups)
	}
	for _, item := range groups[0].Items {
		if item.Source != model.GroupItemSourceAuto {
			t.Fatalf("created item is not auto: %+v", item)
		}
	}
}

func TestAutoGroupValidation(t *testing.T) {
	resetAutoGroupTestState(t)
	invalidMode := model.AutoGroupType(99)
	if err := ChannelCreate(&model.Channel{Name: "invalid", AutoGroup: invalidMode}, context.Background()); err == nil {
		t.Fatal("expected invalid channel mode error")
	}
	channel := createAutoGroupTestChannel(t, "duplicate", model.AutoGroupTypeNone, "gpt-4")
	mode := model.AutoGroupTypeExact
	_, err := GroupAutoGroupConfigUpdate(&model.GroupAutoGroupConfigUpdateRequest{Items: []model.GroupAutoGroupSourceUpdateRequest{
		{ChannelID: channel.ID, AutoGroup: &mode},
		{ChannelID: channel.ID, AutoGroup: &mode},
	}}, context.Background())
	if err == nil {
		t.Fatal("expected duplicate channel error")
	}
}

func TestGlobalAutoGroupModeAppliesAsBatchWhenNoItemsProvided(t *testing.T) {
	resetAutoGroupTestState(t)
	one := createAutoGroupTestChannel(t, "one", model.AutoGroupTypeNone, "gpt-4")
	two := createAutoGroupTestChannel(t, "two", model.AutoGroupTypeNone, "claude-3")
	mode := model.AutoGroupTypeFuzzy
	if _, err := GroupAutoGroupConfigUpdate(&model.GroupAutoGroupConfigUpdateRequest{GlobalMode: &mode}, context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, channel := range []*model.Channel{one, two} {
		updated, err := ChannelGet(channel.ID)
		if err != nil {
			t.Fatal(err)
		}
		if updated.AutoGroup != mode {
			t.Fatalf("channel %d mode = %d, want %d", channel.ID, updated.AutoGroup, mode)
		}
	}
}
