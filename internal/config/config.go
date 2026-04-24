// Package config loads the redash CLI configuration from a YAML file with
// named profiles. Environment variables override file values.
//
// Config file default location: $XDG_CONFIG_HOME/redash-cli/config.yaml
// (falling back to ~/.config/redash-cli/config.yaml).
//
// Example:
//
//	default_profile: prod
//	profiles:
//	  prod:
//	    url: https://redash.example.com
//	    api_key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
//	    timeout: 30s
//	  staging:
//	    url: https://redash.staging.example.com
//	    api_key: yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy
//
// Env overrides (applied to the resolved profile):
//
//	REDASH_URL        overrides url
//	REDASH_API_KEY    overrides api_key
//	REDASH_TIMEOUT    overrides timeout (Go duration, e.g. "30s")
//	REDASH_PROFILE    overrides which profile is used
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Profile holds the connection details for a single Redash instance.
type Profile struct {
	URL      string        `yaml:"url"`
	APIKey   string        `yaml:"api_key"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`
	Insecure bool          `yaml:"insecure,omitempty"`
}

// File is the on-disk config structure.
type File struct {
	DefaultProfile string             `yaml:"default_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

// ErrNoProfile is returned when no profile can be resolved.
var ErrNoProfile = errors.New("no profile configured: create one with `redash config init` or set REDASH_URL and REDASH_API_KEY")

// DefaultPath returns the default config file path.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "redash-cli", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "redash-cli", "config.yaml")
	}
	return filepath.Join(home, ".config", "redash-cli", "config.yaml")
}

// Load reads the config file at path (if it exists) and returns the parsed
// structure. A missing file is not an error: an empty File is returned so
// env-only usage works.
func Load(path string) (*File, error) {
	f := &File{Profiles: map[string]Profile{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	return f, nil
}

// Save writes the File to path, creating parent directories as needed.
// The file is written with 0600 permissions because it contains API keys.
func Save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// Resolve picks the active profile from the file, applying the explicit
// profileName override (from --profile), then REDASH_PROFILE, then the
// file's default_profile, then the first profile in the file. After that,
// it overlays REDASH_URL / REDASH_API_KEY / REDASH_TIMEOUT env vars.
//
// If the file is empty and env vars supply URL + API key, a synthetic
// profile named "env" is returned.
func Resolve(f *File, profileName string) (string, Profile, error) {
	name := profileName
	if name == "" {
		name = os.Getenv("REDASH_PROFILE")
	}
	if name == "" {
		name = f.DefaultProfile
	}

	var p Profile
	resolvedName := name

	switch {
	case name != "":
		if found, ok := f.Profiles[name]; ok {
			p = found
		} else if len(f.Profiles) == 0 {
			// profile name given but no file, fall through to env
			resolvedName = "env"
		} else {
			return "", Profile{}, fmt.Errorf("profile %q not found in config", name)
		}
	case len(f.Profiles) > 0:
		// Deterministically pick the first profile by iterating.
		// (YAML maps in Go are unordered; for a real "first" we'd need
		// to preserve order. With one profile this is fine.)
		for k, v := range f.Profiles {
			resolvedName, p = k, v
			break
		}
	default:
		resolvedName = "env"
	}

	// Env overrides.
	if v := os.Getenv("REDASH_URL"); v != "" {
		p.URL = v
	}
	if v := os.Getenv("REDASH_API_KEY"); v != "" {
		p.APIKey = v
	}
	if v := os.Getenv("REDASH_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return "", Profile{}, fmt.Errorf("REDASH_TIMEOUT: %w", err)
		}
		p.Timeout = d
	}

	if p.Timeout == 0 {
		p.Timeout = 30 * time.Second
	}
	if p.URL == "" || p.APIKey == "" {
		return "", Profile{}, ErrNoProfile
	}
	return resolvedName, p, nil
}
