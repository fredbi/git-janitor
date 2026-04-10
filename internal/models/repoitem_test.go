package models

import "testing"

func TestRepoItem_Title(t *testing.T) {
	cases := []struct {
		name string
		item RepoItem
		want string
	}{
		{
			name: "git repo",
			item: RepoItem{Name: "api", IsGit: true},
			want: "api",
		},
		{
			name: "namespaced git repo title is leaf only",
			item: RepoItem{Name: "api", Namespace: "group/backend", IsGit: true},
			want: "api",
		},
		{
			name: "non-git appends suffix",
			item: RepoItem{Name: "api", IsGit: false},
			want: "api (not git)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.item.Title(); got != c.want {
				t.Errorf("Title() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRepoItem_FilterValue(t *testing.T) {
	cases := []struct {
		name string
		item RepoItem
		want string
	}{
		{
			name: "flat repo filters by name",
			item: RepoItem{Name: "api"},
			want: "api",
		},
		{
			name: "namespaced repo filters by full path",
			item: RepoItem{Name: "api", Namespace: "group/backend"},
			want: "group/backend/api",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.item.FilterValue(); got != c.want {
				t.Errorf("FilterValue() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRepoItem_Depth(t *testing.T) {
	cases := []struct {
		ns   string
		want int
	}{
		{"", 0},
		{"group", 1},
		{"group/sub", 2},
		{"group/sub/leaf", 3},
	}

	for _, c := range cases {
		t.Run(c.ns, func(t *testing.T) {
			item := RepoItem{Name: "x", Namespace: c.ns}
			if got := item.Depth(); got != c.want {
				t.Errorf("Depth(%q) = %d, want %d", c.ns, got, c.want)
			}
		})
	}
}

func TestRepoItem_DisplayKey(t *testing.T) {
	cases := []struct {
		name string
		item RepoItem
		want string
	}{
		{
			name: "flat lowercased",
			item: RepoItem{Name: "API"},
			want: "api",
		},
		{
			name: "namespaced lowercased",
			item: RepoItem{Name: "API", Namespace: "MyGroup/Backend"},
			want: "mygroup/backend/api",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.item.DisplayKey(); got != c.want {
				t.Errorf("DisplayKey() = %q, want %q", got, c.want)
			}
		})
	}
}
