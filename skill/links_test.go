package skill_test

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/StevenACoffman/skillet/skill"
)

func TestBrokenLinks(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"SKILL.md":                 {Data: []byte("x")},
		"references/guide.md":      {Data: []byte("x")},
		"scripts/current-state.sh": {Data: []byte("x")},
		"with space.md":            {Data: []byte("x")},
	}
	cases := map[string]struct {
		body string
		want []string
	}{
		"existing file":         {body: "See [guide](references/guide.md)."},
		"existing directory":    {body: "See [refs](references/)."},
		"dot-slash prefix":      {body: "[s](./scripts/current-state.sh)"},
		"fragment on a file":    {body: "[g](references/guide.md#setup)"},
		"percent-encoded space": {body: "[w](with%20space.md)"},
		"missing file": {
			body: "[g](references/gone.md)",
			want: []string{"references/gone.md"},
		},
		"missing image":          {body: "![d](diagram.png)", want: []string{"diagram.png"}},
		"reported once":          {body: "[a](x.md) [b](x.md)", want: []string{"x.md"}},
		"URL is not judged":      {body: "[u](https://example.com/x.md) [m](mailto:a@b.c)"},
		"fragment only":          {body: "[h](#heading)"},
		"absolute path":          {body: "[a](/etc/passwd)"},
		"home path":              {body: "[h](~/notes.md)"},
		"escaping path":          {body: "[e](../other-skill/SKILL.md)"},
		"link in a code span":    {body: "`[g](missing.md)`"},
		"link in a fenced block": {body: "```\n[g](missing.md)\n```\n"},
		"several in order": {
			body: "[b](b.md) [ok](references/guide.md) [a](a.md)",
			want: []string{"b.md", "a.md"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := skill.BrokenLinks(fsys, tc.body); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("BrokenLinks = %q, want %q", got, tc.want)
			}
		})
	}
}
