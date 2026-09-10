package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSiteFormats(t *testing.T) {
	for _, tt := range []struct{ name, data string }{
		{"yaml", "- name: Example\n  url: https://example.test/{username}\n  check_type: message\n  absence_strs: [missing]\n"},
		{"native-json", `[{"name":"Example","url":"https://example.test/{username}","check_type":"message","absence_strs":["missing"]}]`},
		{"native-map", `{"Example":{"url":"https://example.test/{username}","check_type":"message","absence_strs":["missing"]}}`},
		{"wrapped-native-array", `{"sites":[{"name":"Example","url":"https://example.test/{username}","check_type":"message","absence_strs":["missing"]}]}`},
		{"sherlock", `{"$schema":"schema.json","Example":{"url":"https://example.test/{}","errorType":"message","errorMsg":"missing"}}`},
		{"maigret", `{"sites":{"Example":{"url":"https://example.test/{username}","checkType":"message","absenceStrs":["missing"]},"disabled":{"url":"https://disabled.test/","disabled":true}}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sites")
			if err := os.WriteFile(path, []byte(tt.data), 0600); err != nil {
				t.Fatal(err)
			}
			sites, err := LoadSites(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(sites) != 1 {
				t.Fatalf("sites: %+v", sites)
			}
			s := sites[0]
			if s.Name != "Example" || s.URL != "https://example.test/{username}" || s.CheckType != "message" || len(s.AbsenceStrs) != 1 || s.AbsenceStrs[0] != "missing" {
				t.Fatalf("lost site semantics: %+v", s)
			}
		})
	}
	if sites, err := LoadSites("../../configs/sites.yaml"); err != nil || len(sites) != 20 {
		t.Fatal("default database", len(sites), err)
	}
}
func TestMaigretConversionFields(t *testing.T) {
	data := `{"sites":{"Example":{"url":"https://example.test/{username}","checkType":"message","presenceStrs":["one"],"presenseStrs":["two"],"requestMethod":"POST","requestPayload":{"user":"{username}"},"headers":{"X-Test":"value"},"weight":25,"follow_redirects":true}}}`
	path := filepath.Join(t.TempDir(), "sites.json")
	_ = os.WriteFile(path, []byte(data), 0600)
	sites, err := LoadSites(path)
	if err != nil || len(sites) != 1 {
		t.Fatal(err)
	}
	s := sites[0]
	if len(s.PresenceStrs) != 2 || s.RequestMethod != "POST" || s.RequestPayload == "" || s.Weight != 25 || !s.FollowRedirects || s.Headers["X-Test"] != "value" {
		t.Fatalf("lost fields: %+v", s)
	}
}
func TestServices(t *testing.T) {
	cfg, err := LoadServices("../../configs/services.yaml")
	if err != nil || cfg.Services["hibp"].Enabled || cfg.Services["hibp"].APIKeyEnv != "HIBP_API_KEY" {
		t.Fatal("example config", err)
	}
	path := filepath.Join(t.TempDir(), "services.yaml")
	for _, data := range []string{"services:\n  example:\n    api_key: secret-value\n", "services:\n  example:\n    api_key_env: invalid-name\n", "services: {}\n---\nservices: {}"} {
		_ = os.WriteFile(path, []byte(data), 0600)
		if _, err := LoadServices(path); err == nil {
			t.Fatal("unsafe or invalid config accepted")
		}
	}
}
