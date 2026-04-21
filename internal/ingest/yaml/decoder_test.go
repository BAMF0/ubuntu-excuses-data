package ingest

import (
	"bytes"
	_ "embed"
	"testing"
	"time"
)

//go:embed testdata/base-files_excuses.yaml
var baseFilesExcusesYAML []byte

//go:embed testdata/update-excuse-bugs.yaml
var updateExcuseBugsYAML []byte

// mustDecode decodes YAML data and fails the test immediately on error.
func mustDecode(t *testing.T, data []byte) *ExcusesFile {
	t.Helper()
	f, err := ReadExcusesYaml(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadExcusesYaml: %v", err)
	}
	return f
}

func TestReadBaseFilesYaml(t *testing.T) {
	excuses := mustDecode(t, baseFilesExcusesYAML)
	if len(excuses.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(excuses.Sources))
	}

	source := excuses.Sources[0]
	if source.SourcePackage != "base-files" {
		t.Errorf("SourcePackage = %q, want %q", source.SourcePackage, "base-files")
	}
	if len(source.Reason) != 1 || source.Reason[0] != "autopkgtest" {
		t.Errorf("Reason = %v, want [autopkgtest]", source.Reason)
	}
	if source.MigrationPolicyVerdict != "REJECTED_TEMPORARILY" {
		t.Errorf("MigrationPolicyVerdict = %q, want %q", source.MigrationPolicyVerdict, "REJECTED_TEMPORARILY")
	}
	if source.Maintainer != "Ubuntu Developers" {
		t.Errorf("Maintainer = %q, want %q", source.Maintainer, "Ubuntu Developers")
	}
}

func TestGeneratedDate(t *testing.T) {
	excuses := mustDecode(t, baseFilesExcusesYAML)
	want := time.Date(2026, 4, 20, 12, 53, 48, 193165000, time.UTC)
	if !excuses.GeneratedDate.Equal(want) {
		t.Errorf("GeneratedDate = %v, want %v", excuses.GeneratedDate, want)
	}
}

func TestSourceScalarFields(t *testing.T) {
	source := mustDecode(t, baseFilesExcusesYAML).Sources[0]

	t.Run("ItemName", func(t *testing.T) {
		if source.ItemName != "base-files" {
			t.Errorf("got %q, want %q", source.ItemName, "base-files")
		}
	})
	t.Run("NewVersion", func(t *testing.T) {
		if source.NewVersion != "14ubuntu6" {
			t.Errorf("got %q, want %q", source.NewVersion, "14ubuntu6")
		}
	})
	t.Run("OldVersion", func(t *testing.T) {
		if source.OldVersion != "14ubuntu5" {
			t.Errorf("got %q, want %q", source.OldVersion, "14ubuntu5")
		}
	})
	t.Run("IsCandidate", func(t *testing.T) {
		if source.IsCandidate {
			t.Error("got true, want false")
		}
	})
	t.Run("Dependencies_nil", func(t *testing.T) {
		if source.Dependencies != nil {
			t.Errorf("got %v, want nil", source.Dependencies)
		}
	})
}

func TestExcusesSlice(t *testing.T) {
	src := mustDecode(t, baseFilesExcusesYAML).Sources[0]

	if len(src.Excuses) != 6 {
		t.Fatalf("got %d excuses, want 6", len(src.Excuses))
	}
	t.Run("first", func(t *testing.T) {
		want := "Migration status for base-files (14ubuntu5 to 14ubuntu6): Waiting for test results, another package or too young (no action required now - check later)"
		if src.Excuses[0] != want {
			t.Errorf("got %q, want %q", src.Excuses[0], want)
		}
	})
	t.Run("last", func(t *testing.T) {
		want := "Ignoring block request by freeze, due to unblock request by ubuntu-release"
		if src.Excuses[5] != want {
			t.Errorf("got %q, want %q", src.Excuses[5], want)
		}
	})
}

func TestHints(t *testing.T) {
	src := mustDecode(t, baseFilesExcusesYAML).Sources[0]

	if len(src.Hints) != 2 {
		t.Fatalf("got %d hints, want 2", len(src.Hints))
	}
	t.Run("unblock", func(t *testing.T) {
		h := src.Hints[0]
		if h.HintFrom != "ubuntu-release" {
			t.Errorf("HintFrom = %q, want %q", h.HintFrom, "ubuntu-release")
		}
		if h.HintType != "unblock" {
			t.Errorf("HintType = %q, want %q", h.HintType, "unblock")
		}
	})
	t.Run("block", func(t *testing.T) {
		h := src.Hints[1]
		if h.HintFrom != "freeze" {
			t.Errorf("HintFrom = %q, want %q", h.HintFrom, "freeze")
		}
		if h.HintType != "block" {
			t.Errorf("HintType = %q, want %q", h.HintType, "block")
		}
	})
}

func TestAgePolicy(t *testing.T) {
	age := mustDecode(t, baseFilesExcusesYAML).Sources[0].PolicyInfo.Age

	t.Run("AgeRequirement", func(t *testing.T) {
		if age.AgeRequirement != 0 {
			t.Errorf("got %d, want 0", age.AgeRequirement)
		}
	})
	t.Run("CurrentAge", func(t *testing.T) {
		want := 0.04805555555555555
		if age.CurrentAge != want {
			t.Errorf("got %v, want %v", age.CurrentAge, want)
		}
	})
	t.Run("Verdict", func(t *testing.T) {
		if age.Verdict != "PASS" {
			t.Errorf("got %q, want %q", age.Verdict, "PASS")
		}
	})
}

func TestAutopkgtestPolicy(t *testing.T) {
	apt := mustDecode(t, baseFilesExcusesYAML).Sources[0].PolicyInfo.Autopkgtest

	t.Run("Verdict", func(t *testing.T) {
		if apt.Verdict != "REJECTED_TEMPORARILY" {
			t.Errorf("got %q, want %q", apt.Verdict, "REJECTED_TEMPORARILY")
		}
	})
	t.Run("PackageCount", func(t *testing.T) {
		// bash, dbus, lsb-release-minimal, mmdebstrap, pymoc, s-nail, update-motd
		if len(apt.Packages) != 7 {
			t.Errorf("got %d packages, want 7", len(apt.Packages))
		}
	})
	t.Run("ArchCount", func(t *testing.T) {
		if len(apt.Packages["bash/5.3-2ubuntu1"]) != 6 {
			t.Errorf("got %d arches for bash, want 6", len(apt.Packages["bash/5.3-2ubuntu1"]))
		}
	})
	t.Run("PASS_status", func(t *testing.T) {
		result, ok := apt.Packages["bash/5.3-2ubuntu1"]["amd64"]
		if !ok {
			t.Fatal("missing bash/5.3-2ubuntu1 amd64 result")
		}
		if result.Status != "PASS" {
			t.Errorf("Status = %q, want %q", result.Status, "PASS")
		}
		if result.LogURL == nil {
			t.Error("LogURL is nil, want non-nil")
		}
		if result.PkgURL == nil {
			t.Error("PkgURL is nil, want non-nil")
		}
	})
	t.Run("ALWAYSFAIL_status", func(t *testing.T) {
		result, ok := apt.Packages["bash/5.3-2ubuntu1"]["i386"]
		if !ok {
			t.Fatal("missing bash/5.3-2ubuntu1 i386 result")
		}
		if result.Status != "ALWAYSFAIL" {
			t.Errorf("Status = %q, want %q", result.Status, "ALWAYSFAIL")
		}
	})
	t.Run("RUNNING_status", func(t *testing.T) {
		result, ok := apt.Packages["dbus/1.16.2-2ubuntu4"]["amd64"]
		if !ok {
			t.Fatal("missing dbus/1.16.2-2ubuntu4 amd64 result")
		}
		if result.Status != "RUNNING" {
			t.Errorf("Status = %q, want %q", result.Status, "RUNNING")
		}
	})
}

func TestVerdictPolicies(t *testing.T) {
	pi := mustDecode(t, baseFilesExcusesYAML).Sources[0].PolicyInfo

	policies := []struct {
		name    string
		verdict string
	}{
		{"block", pi.Block.Verdict},
		{"block-bugs", pi.BlockBugs.Verdict},
		{"depends", pi.Depends.Verdict},
		{"email", pi.Email.Verdict},
		{"source-ppa", pi.SourcePPA.Verdict},
	}
	for _, p := range policies {
		t.Run(p.name, func(t *testing.T) {
			if p.verdict != "PASS" {
				t.Errorf("got %q, want %q", p.verdict, "PASS")
			}
		})
	}
	t.Run("linux_nil", func(t *testing.T) {
		if pi.Linux != nil {
			t.Errorf("Linux = %v, want nil", pi.Linux)
		}
	})
}

func TestRcBugsPolicy(t *testing.T) {
	rc := mustDecode(t, baseFilesExcusesYAML).Sources[0].PolicyInfo.RcBugs

	t.Run("Verdict", func(t *testing.T) {
		if rc.Verdict != "PASS" {
			t.Errorf("got %q, want %q", rc.Verdict, "PASS")
		}
	})
	t.Run("SharedBugs", func(t *testing.T) {
		if len(rc.SharedBugs) != 0 {
			t.Errorf("got %v, want empty", rc.SharedBugs)
		}
	})
	t.Run("UniqueSourceBugs", func(t *testing.T) {
		if len(rc.UniqueSourceBugs) != 0 {
			t.Errorf("got %v, want empty", rc.UniqueSourceBugs)
		}
	})
	t.Run("UniqueTargetBugs", func(t *testing.T) {
		if len(rc.UniqueTargetBugs) != 0 {
			t.Errorf("got %v, want empty", rc.UniqueTargetBugs)
		}
	})
}

func TestUpdateExcusePolicy(t *testing.T) {
	t.Run("verdict_only", func(t *testing.T) {
		ue := mustDecode(t, baseFilesExcusesYAML).Sources[0].PolicyInfo.UpdateExcuse
		if ue.Verdict != "PASS" {
			t.Errorf("Verdict = %q, want %q", ue.Verdict, "PASS")
		}
		if len(ue.Bugs) != 0 {
			t.Errorf("Bugs = %v, want empty map", ue.Bugs)
		}
	})
	t.Run("with_bugs", func(t *testing.T) {
		ue := mustDecode(t, updateExcuseBugsYAML).Sources[0].PolicyInfo.UpdateExcuse
		if ue.Verdict != "REJECTED_PERMANENTLY" {
			t.Errorf("Verdict = %q, want %q", ue.Verdict, "REJECTED_PERMANENTLY")
		}
		if len(ue.Bugs) != 1 {
			t.Fatalf("got %d bugs, want 1", len(ue.Bugs))
		}
		ts, ok := ue.Bugs["2142117"]
		if !ok {
			t.Fatal("missing bug 2142117")
		}
		if ts != 1771420300 {
			t.Errorf("bug timestamp = %d, want 1771420300", ts)
		}
	})
}
