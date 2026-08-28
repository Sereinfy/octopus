package model

import (
	"fmt"
	"net/url"
	"strconv"
)

type SettingKey string

const (
	SettingKeyProxyURL                       SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval              SettingKey = "stats_save_interval"                // 将统计信息写入数据库的周期(分钟)
	SettingKeyModelInfoUpdateInterval        SettingKey = "model_info_update_interval"         // 模型信息更新间隔(小时)
	SettingKeySyncLLMInterval                SettingKey = "sync_llm_interval"                  // LLM 同步间隔(小时)
	SettingKeyCORSAllowOrigins               SettingKey = "cors_allow_origins"                 // 跨域白名单(逗号分隔, 如 "example.com,example2.com"). 为空不允许跨域, "*"允许所有
	SettingKeyReplaceDeveloperRoleWithSystem SettingKey = "replace_developer_role_with_system" // 将 developer 消息角色替换为 system
	SettingKeyAutoGroupGlobalMode            SettingKey = "auto_group_global_mode"             // 全局自动分组模式（0关闭/1模糊/2精确）
	SettingKeyAutoGroupCreateMissingEnabled  SettingKey = "auto_group_create_missing_enabled"  // 是否自动创建缺失分组
	SettingKeyAutoGroupNormalizeEnabled      SettingKey = "auto_group_normalize_enabled"       // 是否归一化模型名
)

type Setting struct {
	Key   SettingKey `json:"key" gorm:"primaryKey"`
	Value string     `json:"value" gorm:"not null"`
}

func DefaultSettings() []Setting {
	return []Setting{
		{Key: SettingKeyProxyURL, Value: ""},
		{Key: SettingKeyStatsSaveInterval, Value: "10"},                 // 默认10分钟保存一次统计信息
		{Key: SettingKeyCORSAllowOrigins, Value: ""},                    // CORS 默认不允许跨域，设置为 "*" 才允许所有来源
		{Key: SettingKeyModelInfoUpdateInterval, Value: "24"},           // 默认24小时更新一次模型信息
		{Key: SettingKeySyncLLMInterval, Value: "24"},                   // 默认24小时同步一次LLM
		{Key: SettingKeyReplaceDeveloperRoleWithSystem, Value: "false"}, // 默认保留 developer 角色
		{Key: SettingKeyAutoGroupGlobalMode, Value: "0"},
		{Key: SettingKeyAutoGroupCreateMissingEnabled, Value: "false"},
		{Key: SettingKeyAutoGroupNormalizeEnabled, Value: "false"},
	}
}

func (s *Setting) Validate() error {
	switch s.Key {
	case SettingKeyModelInfoUpdateInterval, SettingKeySyncLLMInterval:
		_, err := strconv.Atoi(s.Value)
		if err != nil {
			return fmt.Errorf("model info update interval must be an integer")
		}
		return nil
	case SettingKeyReplaceDeveloperRoleWithSystem:
		if _, err := strconv.ParseBool(s.Value); err != nil {
			return fmt.Errorf("replace developer role with system must be a boolean")
		}
		return nil
	case SettingKeyAutoGroupGlobalMode:
		if _, ok := ParseAutoGroupSettingValue(s.Value); !ok {
			return fmt.Errorf("auto group global mode must be one of 0, 1, 2, true, false")
		}
		return nil
	case SettingKeyAutoGroupCreateMissingEnabled, SettingKeyAutoGroupNormalizeEnabled:
		if _, err := strconv.ParseBool(s.Value); err != nil {
			return fmt.Errorf("auto group setting must be a boolean")
		}
		return nil
	case SettingKeyProxyURL:
		if s.Value == "" {
			return nil
		}
		parsedURL, err := url.Parse(s.Value)
		if err != nil {
			return fmt.Errorf("proxy URL is invalid: %w", err)
		}
		validSchemes := map[string]bool{
			"http":   true,
			"https":  true,
			"socks5": true,
		}
		if !validSchemes[parsedURL.Scheme] {
			return fmt.Errorf("proxy URL scheme must be http, https, socks, or socks5")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("proxy URL must have a host")
		}
		return nil
	}

	return nil
}
