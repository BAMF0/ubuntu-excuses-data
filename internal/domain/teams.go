package domain

import (
	"encoding/json"
	"fmt"
	"os"
)

// TeamMappings maps each source package name to the team responsible for it.
//
// Constructed from package_team_mappings.json, which stores the inverse relation
// (team → []package); this type inverts that for O(1) lookup by package name.
// The special "unsubscribed" key is excluded so packages without a team are
// represented as an empty string.
type TeamMappings map[string]string

// LoadTeamMappings reads the JSON file at path and returns a TeamMappings.
// The file format is {"team-name": ["pkg1", "pkg2", ...], ...}.
func LoadTeamMappings(path string) (m TeamMappings, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", path, closeErr)
		}
	}()
	var raw map[string][]string
	if err = json.NewDecoder(f).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return make(TeamMappings, len(raw)*8), err
}

// Team returns the team responsible for the given package, or empty string if unknown.
func (t TeamMappings) Team(pkg string) string {
	return t[pkg]
}
