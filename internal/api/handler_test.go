package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BAMF0/ubuntu-excuses-data/internal/domain"
)

// testExcuses builds a small Excuses dataset for handler tests.
func testExcuses() *domain.Excuses {
	b := domain.NewBuilder(4)
	b.Add(&domain.Source{
		SourcePackage: "bash",
		ComponentID:   b.InternComponent("main"),
		MaintainerID:  b.InternMaintainer("ubuntu-devel@lists.ubuntu.com"),
		VerdictID:     b.InternVerdict("PASS"),
		IsCandidate:   true,
		ItemName:      "bash/5.3-1",
		OldVersion:    "5.2-1",
		NewVersion:    "5.3-1",
		Excuse: domain.Excuse{
			Status: domain.StatusWillAttempt,
			Detail: "",
			Info:   []string{"0 days old"},
		},
		PolicyInfo: domain.PolicyInfo{
			Age: domain.AgePolicy{AgeRequirement: 5, CurrentAge: 3, Verdict: "PASS"},
			Autopkgtest: domain.AutopkgtestPolicy{
				Verdict: "PASS",
				Packages: map[string]map[domain.ArchID]domain.AutopkgtestResult{
					"bash/5.3-1": {
						b.InternArch("amd64"): {
							StatusID: b.InternStatus("PASS"),
						},
					},
				},
			},
		},
	})
	b.Add(&domain.Source{
		SourcePackage: "zlib",
		ComponentID:   b.InternComponent("main"),
		MaintainerID:  b.InternMaintainer("ubuntu-devel@lists.ubuntu.com"),
		VerdictID:     b.InternVerdict("REJECTED_PERMANENTLY"),
		IsCandidate:   true,
		ItemName:      "zlib/1.3-1",
		OldVersion:    "1.2-1",
		NewVersion:    "1.3-1",
		Excuse: domain.Excuse{
			Status: domain.StatusBlocked,
			Detail: "introduces a regression",
		},
		PolicyInfo: domain.PolicyInfo{
			Age: domain.AgePolicy{AgeRequirement: 10, CurrentAge: 7, Verdict: "PASS"},
		},
	})
	b.Add(&domain.Source{
		SourcePackage: "vim",
		ComponentID:   b.InternComponent("universe"),
		MaintainerID:  b.InternMaintainer("pkg-vim@lists.alioth.debian.org"),
		VerdictID:     b.InternVerdict("PASS"),
		IsCandidate:   false,
		ItemName:      "vim/9.1-1",
		OldVersion:    "9.0-1",
		NewVersion:    "9.1-1",
		Excuse: domain.Excuse{
			Status: domain.StatusWaiting,
		},
		PolicyInfo: domain.PolicyInfo{
			Age: domain.AgePolicy{AgeRequirement: 5, CurrentAge: 1, Verdict: "REJECTED_TEMPORARILY"},
		},
	})
	return b.Build(time.Date(2025, 4, 20, 12, 0, 0, 0, time.UTC))
}

func newTestServer(e *domain.Excuses) *httptest.Server {
	mux := http.NewServeMux()
	RegisterRoutes(mux, e)
	return httptest.NewServer(mux)
}

func TestGetMeta(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/meta")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var meta MetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if meta.TotalSources != 3 {
		t.Errorf("TotalSources = %d, want 3", meta.TotalSources)
	}
	if meta.TotalCandidates != 2 {
		t.Errorf("TotalCandidates = %d, want 2", meta.TotalCandidates)
	}
	if meta.GeneratedDate != "2025-04-20T12:00:00Z" {
		t.Errorf("GeneratedDate = %q, want 2025-04-20T12:00:00Z", meta.GeneratedDate)
	}
}

func TestListSources_All(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 3 {
		t.Errorf("Total = %d, want 3", list.Total)
	}
	if len(list.Sources) != 3 {
		t.Errorf("len(Sources) = %d, want 3", len(list.Sources))
	}
	// Verify alphabetical order.
	if list.Sources[0].SourcePackage != "bash" {
		t.Errorf("first source = %q, want bash", list.Sources[0].SourcePackage)
	}
	if list.Sources[2].SourcePackage != "zlib" {
		t.Errorf("last source = %q, want zlib", list.Sources[2].SourcePackage)
	}
}

func TestListSources_Pagination(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?offset=1&limit=1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 3 {
		t.Errorf("Total = %d, want 3", list.Total)
	}
	if len(list.Sources) != 1 {
		t.Fatalf("len(Sources) = %d, want 1", len(list.Sources))
	}
	if list.Sources[0].SourcePackage != "vim" {
		t.Errorf("source = %q, want vim (alphabetical index 1)", list.Sources[0].SourcePackage)
	}
}

func TestListSources_PaginationBeyondEnd(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?offset=100")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Sources) != 0 {
		t.Errorf("len(Sources) = %d, want 0", len(list.Sources))
	}
}

func TestListSources_FilterByComponent(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?component=universe")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("Total = %d, want 1", list.Total)
	}
	if list.Sources[0].SourcePackage != "vim" {
		t.Errorf("source = %q, want vim", list.Sources[0].SourcePackage)
	}
}

func TestListSources_FilterByVerdict(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?verdict=REJECTED_PERMANENTLY")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("Total = %d, want 1", list.Total)
	}
	if list.Sources[0].SourcePackage != "zlib" {
		t.Errorf("source = %q, want zlib", list.Sources[0].SourcePackage)
	}
}

func TestListSources_FilterByMigrationStatus(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?status=BLOCKED")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("Total = %d, want 1", list.Total)
	}
	if list.Sources[0].SourcePackage != "zlib" {
		t.Errorf("source = %q, want zlib", list.Sources[0].SourcePackage)
	}
}

func TestListSources_FilterNoMatch(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?component=nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 0 {
		t.Errorf("Total = %d, want 0", list.Total)
	}
}

func TestGetSource(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources/bash")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var src SourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&src); err != nil {
		t.Fatal(err)
	}
	if src.SourcePackage != "bash" {
		t.Errorf("SourcePackage = %q, want bash", src.SourcePackage)
	}
	if src.Component != "main" {
		t.Errorf("Component = %q, want main", src.Component)
	}
	if src.MigrationStatus != "WILL_ATTEMPT" {
		t.Errorf("MigrationStatus = %q, want WILL_ATTEMPT", src.MigrationStatus)
	}
}

func TestGetSource_NotFound(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetSourceAutopkgtest(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources/bash/autopkgtest")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var apt AutopkgtestPolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&apt); err != nil {
		t.Fatal(err)
	}
	if apt.Verdict != "PASS" {
		t.Errorf("Verdict = %q, want PASS", apt.Verdict)
	}
	archResults, ok := apt.Packages["bash/5.3-1"]
	if !ok {
		t.Fatal("missing package bash/5.3-1")
	}
	res, ok := archResults["amd64"]
	if !ok {
		t.Fatal("missing arch amd64")
	}
	if res.Status != "PASS" {
		t.Errorf("Status = %q, want PASS", res.Status)
	}
}

func TestGetSourceAutopkgtest_NotFound(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources/nonexistent/autopkgtest")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListSources_CombinedFilters(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	// main + PASS → bash and vim? No, vim is universe. So just bash.
	resp, err := http.Get(srv.URL + "/sources?component=main&verdict=PASS")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("Total = %d, want 1", list.Total)
	}
	if list.Sources[0].SourcePackage != "bash" {
		t.Errorf("source = %q, want bash", list.Sources[0].SourcePackage)
	}
}

func TestListSources_SortByAge(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	// Ages: vim=1, bash=3, zlib=7 → ascending by age.
	resp, err := http.Get(srv.URL + "/sources?sort=age")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Sort != "age" {
		t.Errorf("Sort = %q, want age", list.Sort)
	}
	if list.Order != "asc" {
		t.Errorf("Order = %q, want asc", list.Order)
	}
	if len(list.Sources) != 3 {
		t.Fatalf("len(Sources) = %d, want 3", len(list.Sources))
	}
	want := []string{"vim", "bash", "zlib"}
	for i, w := range want {
		if list.Sources[i].SourcePackage != w {
			t.Errorf("Sources[%d] = %q, want %q", i, list.Sources[i].SourcePackage, w)
		}
	}
}

func TestListSources_SortByAgeDesc(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?sort=age&order=desc")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Sort != "age" {
		t.Errorf("Sort = %q, want age", list.Sort)
	}
	if list.Order != "desc" {
		t.Errorf("Order = %q, want desc", list.Order)
	}
	if len(list.Sources) != 3 {
		t.Fatalf("len(Sources) = %d, want 3", len(list.Sources))
	}
	want := []string{"zlib", "bash", "vim"}
	for i, w := range want {
		if list.Sources[i].SourcePackage != w {
			t.Errorf("Sources[%d] = %q, want %q", i, list.Sources[i].SourcePackage, w)
		}
	}
}

func TestListSources_SortByNameDesc(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources?sort=name&order=desc")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Sources) != 3 {
		t.Fatalf("len(Sources) = %d, want 3", len(list.Sources))
	}
	want := []string{"zlib", "vim", "bash"}
	for i, w := range want {
		if list.Sources[i].SourcePackage != w {
			t.Errorf("Sources[%d] = %q, want %q", i, list.Sources[i].SourcePackage, w)
		}
	}
}

func TestListSources_DefaultSortInResponse(t *testing.T) {
	srv := newTestServer(testExcuses())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sources")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var list SourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Sort != "name" {
		t.Errorf("Sort = %q, want name", list.Sort)
	}
	if list.Order != "asc" {
		t.Errorf("Order = %q, want asc", list.Order)
	}
}
