package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	f, err := Load(filepath.Join(dir, "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if len(f.Profiles) != 0 {
		t.Errorf("expected empty Profiles, got %v", f.Profiles)
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	in := &File{
		DefaultProfile: "prod",
		Profiles: map[string]Profile{
			"prod":    {URL: "https://redash.example.com", APIKey: "abc", Timeout: 10 * time.Second},
			"staging": {URL: "https://staging.example.com", APIKey: "def"},
		},
	}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 perms (contains secrets), got %o", info.Mode().Perm())
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.DefaultProfile != "prod" {
		t.Errorf("default profile: %q", out.DefaultProfile)
	}
	if got := out.Profiles["prod"].URL; got != "https://redash.example.com" {
		t.Errorf("prod URL: %q", got)
	}
	if got := out.Profiles["prod"].Timeout; got != 10*time.Second {
		t.Errorf("prod timeout: %v", got)
	}
}

func TestResolveUsesExplicitProfile(t *testing.T) {
	clearEnv(t)
	f := &File{
		DefaultProfile: "prod",
		Profiles: map[string]Profile{
			"prod":    {URL: "https://prod", APIKey: "p"},
			"staging": {URL: "https://staging", APIKey: "s"},
		},
	}
	name, p, err := Resolve(f, "staging")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if name != "staging" || p.URL != "https://staging" || p.APIKey != "s" {
		t.Errorf("got name=%q url=%q key=%q", name, p.URL, p.APIKey)
	}
}

func TestResolveEnvProfileOverride(t *testing.T) {
	clearEnv(t)
	t.Setenv("REDASH_PROFILE", "staging")
	f := &File{
		DefaultProfile: "prod",
		Profiles: map[string]Profile{
			"prod":    {URL: "https://prod", APIKey: "p"},
			"staging": {URL: "https://staging", APIKey: "s"},
		},
	}
	name, _, err := Resolve(f, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if name != "staging" {
		t.Errorf("expected staging, got %q", name)
	}
}

func TestResolveEnvUrlAndKeyOverride(t *testing.T) {
	clearEnv(t)
	t.Setenv("REDASH_URL", "https://env")
	t.Setenv("REDASH_API_KEY", "env-key")
	t.Setenv("REDASH_TIMEOUT", "5s")
	f := &File{
		DefaultProfile: "prod",
		Profiles: map[string]Profile{
			"prod": {URL: "https://prod", APIKey: "p", Timeout: 30 * time.Second},
		},
	}
	_, p, err := Resolve(f, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.URL != "https://env" || p.APIKey != "env-key" || p.Timeout != 5*time.Second {
		t.Errorf("got %+v", p)
	}
}

func TestResolveEnvOnly(t *testing.T) {
	clearEnv(t)
	t.Setenv("REDASH_URL", "https://env")
	t.Setenv("REDASH_API_KEY", "env-key")
	f := &File{Profiles: map[string]Profile{}}
	name, p, err := Resolve(f, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if name != "env" || p.URL != "https://env" || p.APIKey != "env-key" {
		t.Errorf("got name=%q %+v", name, p)
	}
	if p.Timeout == 0 {
		t.Errorf("expected default timeout, got 0")
	}
}

func TestResolveMissingRaisesErrNoProfile(t *testing.T) {
	clearEnv(t)
	f := &File{Profiles: map[string]Profile{}}
	_, _, err := Resolve(f, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	clearEnv(t)
	f := &File{
		Profiles: map[string]Profile{"prod": {URL: "https://x", APIKey: "k"}},
	}
	_, _, err := Resolve(f, "nope")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"REDASH_URL", "REDASH_API_KEY", "REDASH_TIMEOUT", "REDASH_PROFILE"} {
		t.Setenv(k, "")
		// t.Setenv("", "") doesn't delete, so explicitly unset via Unsetenv:
		os.Unsetenv(k)
	}
}
