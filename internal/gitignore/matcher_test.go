// Test cases ported from go-git: github.com/go-git/go-git/v5 v5.19.1,
// plumbing/format/gitignore/matcher_test.go. Licensed under the Apache
// License 2.0; see LICENSE.

package gitignore

import "testing"

func TestMatcher_Match(t *testing.T) {
	t.Parallel()

	m := NewMatcher([]Pattern{
		ParsePattern("**/middle/v[uo]l?ano", nil),
		ParsePattern("!volcano", nil),
	})

	if got := m.Match([]string{"head", "middle", "vulkano"}, false); got != true {
		t.Errorf("Match(vulkano) = %t, want true", got)
	}
	if got := m.Match([]string{"head", "middle", "volcano"}, false); got != false {
		t.Errorf("Match(volcano) = %t, want false (negated)", got)
	}
}
