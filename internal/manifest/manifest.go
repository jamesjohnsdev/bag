package manifest

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Commands map[string]string
	Binaries map[string]BinaryEntry
}

type BinaryEntry struct {
	Source  string `toml:"source"`
	Version string `toml:"version"`
	Type    string `toml:"type"` // "binary" | "script"
}

func Parse(path string) (*Manifest, error) {
	var raw map[string]any
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, err
	}
	man := &Manifest{
		Commands: make(map[string]string),
		Binaries: make(map[string]BinaryEntry),
	}
	if cmds, ok := raw["commands"].(map[string]any); ok {
		for i, val := range cmds {
			if s, ok := val.(string); ok {
				man.Commands[i] = s
			}
		}
	}
	for i, val := range raw {
		if i == "commands" {
			continue
		}
		m, ok := val.(map[string]any)
		if !ok {
			continue
		}
		// Expected behaviour is to default to binary
		entry := BinaryEntry{Type: "binary"}
		if src, ok := m["source"].(string); ok {
			entry.Source = src
		}
		if ver, ok := m["version"].(string); ok {
			entry.Version = ver
		}
		if typ, ok := m["type"].(string); ok {
			entry.Type = typ
		}
		man.Binaries[i] = entry
	}
	return man, nil
}

// Write creates or overrides a toml manifest file
func Write(path string, man *Manifest) error {
	raw := make(map[string]any)
	if len(man.Commands) > 0 {
		raw["commands"] = man.Commands
	}
	for i, bin := range man.Binaries {
		raw[i] = bin
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	return toml.NewEncoder(file).Encode(raw)
}

// Add updates a manifest file with an additional binary entry
func AddBinary(manifestPath, name string, entry BinaryEntry) error {
	manifest, err := Parse(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	// currently this will overrite similar named items
	// should consider prompt/check
	manifest.Binaries[name] = entry
	return Write(manifestPath, manifest)
}
