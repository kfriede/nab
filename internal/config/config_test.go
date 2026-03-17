package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Budget != "last-used" {
		t.Errorf("Budget = %q, want %q", cfg.Budget, "last-used")
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if cfg.Profile != "" {
		t.Errorf("Profile = %q, want empty", cfg.Profile)
	}
	if cfg.Verbose {
		t.Error("Verbose should be false")
	}
	if cfg.Debug {
		t.Error("Debug should be false")
	}
}

func TestConfigDirNABConfig(t *testing.T) {
	t.Setenv("NAB_CONFIG", "/custom/nab/config")
	t.Setenv("XDG_CONFIG_HOME", "/should/be/ignored")

	got := configDir()
	if got != "/custom/nab/config" {
		t.Errorf("configDir() = %q, want %q", got, "/custom/nab/config")
	}
}

func TestConfigDirXDG(t *testing.T) {
	t.Setenv("NAB_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/home")

	got := configDir()
	want := filepath.Join("/xdg/home", "nab")
	if got != want {
		t.Errorf("configDir() = %q, want %q", got, want)
	}
}

func TestConfigDirDefault(t *testing.T) {
	t.Setenv("NAB_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	got := configDir()
	want := filepath.Join(home, ".config", "nab")
	if got != want {
		t.Errorf("configDir() = %q, want %q", got, want)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NAB_CONFIG", tmp)
	viper.Reset()

	cfg := &Config{Budget: "my-budget"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Budget != "my-budget" {
		t.Errorf("Budget = %q, want %q", loaded.Budget, "my-budget")
	}
}

func TestSaveProfile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NAB_CONFIG", tmp)
	viper.Reset()

	cfg := &Config{Budget: "work-budget", Profile: "work"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	expectedPath := filepath.Join(tmp, "work.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected profile config at %s, but file does not exist", expectedPath)
	}

	// Default config.yaml should not exist
	defaultPath := filepath.Join(tmp, "config.yaml")
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Errorf("expected no config.yaml when saving a profile, but it exists")
	}
}

func TestSavePermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NAB_CONFIG", tmp)
	viper.Reset()

	cfg := &Config{Budget: "test"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, "config.yaml"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("permissions = %04o, want 0600", perm)
	}
}

func TestEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	nested := filepath.Join(tmp, "sub", "dir")
	t.Setenv("NAB_CONFIG", nested)

	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", nested)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NAB_CONFIG", tmp)
	viper.Reset()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with no config file should not error, got: %v", err)
	}
	if cfg.Budget != "last-used" {
		t.Errorf("Budget = %q, want %q", cfg.Budget, "last-used")
	}
}

func TestKeyringUser(t *testing.T) {
	tests := []struct {
		profile string
		want    string
	}{
		{"", "default"},
		{"work", "work"},
		{"personal", "personal"},
	}
	for _, tt := range tests {
		got := keyringUser(tt.profile)
		if got != tt.want {
			t.Errorf("keyringUser(%q) = %q, want %q", tt.profile, got, tt.want)
		}
	}
}
