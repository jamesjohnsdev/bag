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

type VersionEntry struct {
	Source string `toml:"source"`
}

// TODO: change to Goa and set up sum type for `Type` field
type BinaryEntry struct {
	Type     string                  `toml:"type"` // "binary" | "script"
	Active   string                  `toml:"active"`
	Versions map[string]VersionEntry `toml:"versions"`
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
	if commands, ok := raw["commands"].(map[string]any); ok {
		for cmdName, cmdVal := range commands {
			if cmdStr, ok := cmdVal.(string); ok {
				man.Commands[cmdName] = cmdStr
			}
		}
	}
	for binName, binVal := range raw {
		if binName == "commands" {
			continue
		}
		binFields, ok := binVal.(map[string]any)
		if !ok {
			continue
		}
		// Expected behaviour is to default to binary
		entry := BinaryEntry{Type: "binary", Versions: make(map[string]VersionEntry)}
		if typ, ok := binFields["type"].(string); ok {
			entry.Type = typ
		}
		if active, ok := binFields["active"].(string); ok {
			entry.Active = active
		}
		if versions, ok := binFields["versions"].(map[string]any); ok {
			for versionName, versionVal := range versions {
				versionFields, ok := versionVal.(map[string]any)
				if !ok {
					continue
				}
				var versionEntry VersionEntry
				if src, ok := versionFields["source"].(string); ok {
					versionEntry.Source = src
				}
				entry.Versions[versionName] = versionEntry
			}
		}
		man.Binaries[binName] = entry
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

// RemoveBinary updates a manifest file, removing a binary entry
func RemoveBinary(manifestPath, name string) error {
	manifest, err := Parse(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	delete(manifest.Binaries, name)
	return Write(manifestPath, manifest)
}
