package repos

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/fredbi/git-janitor/internal/models"
)

// itemKey reduces a list.Item to a printable form for assertion: headers
// render as "  group/" prefixed by depth indentation, repos render as
// their full namespace+name.
func itemKey(it list.Item) string {
	switch v := it.(type) {
	case groupHeaderItem:
		return prefix(v.Depth) + v.Name + "/"
	case models.RepoItem:
		if v.Namespace != "" {
			return v.Namespace + "/" + v.Name
		}
		return v.Name
	default:
		return "?"
	}
}

func prefix(depth int) string {
	return strings.Repeat(".", depth)
}

func TestBuildDisplayList_FlatRepos(t *testing.T) {
	repos := []models.RepoItem{
		{Name: "alpha"},
		{Name: "beta"},
	}

	got := buildDisplayList(repos, true)
	want := []string{"alpha", "beta"}
	if keys := keysOf(got); !reflect.DeepEqual(keys, want) {
		t.Errorf("got %v, want %v", keys, want)
	}
}

func TestBuildDisplayList_NestedInsertsHeaders(t *testing.T) {
	repos := []models.RepoItem{
		{Name: "api", Namespace: "myorg/backend"},
		{Name: "db", Namespace: "myorg/backend"},
		{Name: "site", Namespace: "myorg/web"},
		{Name: "standalone"},
	}

	got := buildDisplayList(repos, true)

	// Expected layout:
	//   myorg/                <- depth 0 header
	//   .backend/             <- depth 1 header
	//   myorg/backend/api
	//   myorg/backend/db
	//   .web/                 <- depth 1 header (myorg already common)
	//   myorg/web/site
	//   standalone
	want := []string{
		"myorg/",
		".backend/",
		"myorg/backend/api",
		"myorg/backend/db",
		".web/",
		"myorg/web/site",
		"standalone",
	}
	if keys := keysOf(got); !reflect.DeepEqual(keys, want) {
		t.Errorf("got %v, want %v", keys, want)
	}
}

func TestBuildDisplayList_NoHeadersWhenFiltering(t *testing.T) {
	repos := []models.RepoItem{
		{Name: "api", Namespace: "myorg/backend"},
		{Name: "site", Namespace: "myorg/web"},
	}

	got := buildDisplayList(repos, false)
	want := []string{"myorg/backend/api", "myorg/web/site"}
	if keys := keysOf(got); !reflect.DeepEqual(keys, want) {
		t.Errorf("got %v, want %v", keys, want)
	}
}

func TestBuildDisplayList_MixedFlatAndNested(t *testing.T) {
	// Sorted order from the scanner: nested groups before bare top-level.
	repos := []models.RepoItem{
		{Name: "deep", Namespace: "alpha/group"},
		{Name: "top"},
	}

	got := buildDisplayList(repos, true)
	want := []string{
		"alpha/",
		".group/",
		"alpha/group/deep",
		"top",
	}
	if keys := keysOf(got); !reflect.DeepEqual(keys, want) {
		t.Errorf("got %v, want %v", keys, want)
	}
}

func TestBuildDisplayList_EmptyInput(t *testing.T) {
	if got := buildDisplayList(nil, true); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestCommonPrefixLen(t *testing.T) {
	cases := []struct {
		a, b []string
		want int
	}{
		{nil, nil, 0},
		{[]string{"a"}, nil, 0},
		{[]string{"a"}, []string{"a"}, 1},
		{[]string{"a", "b"}, []string{"a", "c"}, 1},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c", "d"}, 3},
	}

	for _, c := range cases {
		if got := commonPrefixLen(c.a, c.b); got != c.want {
			t.Errorf("commonPrefixLen(%v,%v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func keysOf(items []list.Item) []string {
	keys := make([]string, len(items))
	for i, it := range items {
		keys[i] = itemKey(it)
	}

	return keys
}
