package model

import "testing"

func TestReplaceDeveloperRoleWithSystemSetting(t *testing.T) {
	setting := Setting{Key: SettingKeyReplaceDeveloperRoleWithSystem, Value: "true"}
	if err := setting.Validate(); err != nil {
		t.Fatalf("valid boolean setting rejected: %v", err)
	}

	setting.Value = "not-a-bool"
	if err := setting.Validate(); err == nil {
		t.Fatal("invalid boolean setting accepted")
	}

	found := false
	for _, defaultSetting := range DefaultSettings() {
		if defaultSetting.Key == SettingKeyReplaceDeveloperRoleWithSystem {
			found = true
			if defaultSetting.Value != "false" {
				t.Fatalf("default value = %q, want false", defaultSetting.Value)
			}
		}
	}
	if !found {
		t.Fatal("replace developer role setting missing from defaults")
	}
}
