// Package loadsplugin is a conformance fixture: an orchestrator loads a plugin by a
// path it is given.
package loadsplugin

import (
	"fmt"
	"plugin"
)

// Load opens the plugin at path.
func Load(path string) error {
	if _, err := plugin.Open(path); err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	return nil
}
