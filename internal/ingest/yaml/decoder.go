package ingest

import (
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

// ReadExcusesYaml decodes update_excuses.yaml from r into an ExcusesFile.
func ReadExcusesYaml(r io.Reader) (*ExcusesFile, error) {
	var excuses ExcusesFile
	if err := yaml.NewDecoder(r).Decode(&excuses); err != nil {
		return nil, err
	}
	return &excuses, nil
}

// UnmarshalYAML handles the mixed-key autopkgtest map:
//
//	autopkgtest:
//	  verdict: PASS
//	  bash/5.3-2ubuntu1:
//	    amd64: [PASS, <log_url>, <pkg_url>, null, null]
func (a *AutopkgtestPolicy) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]any
	if err := unmarshal(&raw); err != nil {
		return err
	}

	a.Packages = make(map[string]map[string]AutopkgtestResult)

	for key, val := range raw {
		if key == "verdict" {
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("autopkgtest verdict: expected string, got %T", val)
			}
			a.Verdict = s
			continue
		}

		archMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("autopkgtest package %q: expected map of architectures, got %T", key, val)
		}

		pkgResults := make(map[string]AutopkgtestResult)
		for arch, archVal := range archMap {
			items, ok := archVal.([]any)
			if !ok {
				return fmt.Errorf("autopkgtest package %q arch %q: expected sequence, got %T", key, arch, archVal)
			}
			if len(items) < 3 {
				return fmt.Errorf("autopkgtest package %q arch %q: expected at least 3 values, got %d", key, arch, len(items))
			}
			var result AutopkgtestResult
			if s, ok := items[0].(string); ok {
				result.Status = s
			}
			if s, ok := items[1].(string); ok {
				result.LogURL = &s
			}
			if s, ok := items[2].(string); ok {
				result.PkgURL = &s
			}
			pkgResults[arch] = result
		}
		a.Packages[key] = pkgResults
	}

	return nil
}

// UnmarshalYAML handles the mixed-key update-excuse map:
//
//	update-excuse:
//	  verdict: PASS
//	  "2142117": 1771420300
func (u *UpdateExcusePolicy) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]any
	if err := unmarshal(&raw); err != nil {
		return err
	}

	u.Bugs = make(map[string]int64)

	for key, val := range raw {
		if key == "verdict" {
			s, ok := val.(string)
			if !ok {
				return fmt.Errorf("update-excuse verdict: expected string, got %T", val)
			}
			u.Verdict = s
			continue
		}

		switch v := val.(type) {
		case int:
			u.Bugs[key] = int64(v)
		case int64:
			u.Bugs[key] = v
		case uint:
			u.Bugs[key] = int64(v)
		case uint64:
			u.Bugs[key] = int64(v)
		default:
			return fmt.Errorf("update-excuse bug %q: expected integer timestamp, got %T", key, val)
		}
	}

	return nil
}
