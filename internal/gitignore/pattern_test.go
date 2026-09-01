// Test cases ported from go-git: github.com/go-git/go-git/v5 v5.19.1,
// plumbing/format/gitignore/pattern_test.go (originally written against
// gopkg.in/check.v1; rewritten here as table-driven stdlib tests to avoid an
// extra test dependency). Licensed under the Apache License 2.0; see LICENSE.

package gitignore

import "testing"

func TestParsePattern_Match(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		domain  []string
		path    []string
		isDir   bool
		want    MatchResult
	}{
		{"inclusion", "!vul?ano", nil, []string{"value", "vulkano", "tail"}, false, Include},
		{"domainLonger_mismatch", "value", []string{"head", "middle", "tail"}, []string{"head", "middle"}, false, NoMatch},
		{"domainSameLength_mismatch", "value", []string{"head", "middle", "tail"}, []string{"head", "middle", "tail"}, false, NoMatch},
		{"domainMismatch_mismatch", "value", []string{"head", "middle", "tail"}, []string{"head", "middle", "_tail_", "value"}, false, NoMatch},
		{"withDomain", "middle/", []string{"value", "volcano"}, []string{"value", "volcano", "middle", "tail"}, false, Exclude},
		{"onlyMatchInDomain_mismatch", "volcano/", []string{"value", "volcano"}, []string{"value", "volcano", "tail"}, true, NoMatch},
		{"atStart", "value", nil, []string{"value", "tail"}, false, Exclude},
		{"inTheMiddle", "value", nil, []string{"head", "value", "tail"}, false, Exclude},
		{"atEnd", "value", nil, []string{"head", "value"}, false, Exclude},
		{"atStart_dirWanted", "value/", nil, []string{"value", "tail"}, false, Exclude},
		{"inTheMiddle_dirWanted", "value/", nil, []string{"head", "value", "tail"}, false, Exclude},
		{"atEnd_dirWanted", "value/", nil, []string{"head", "value"}, true, Exclude},
		{"atEnd_dirWanted_notADir_mismatch", "value/", nil, []string{"head", "value"}, false, NoMatch},
		{"mismatch", "value", nil, []string{"head", "val", "tail"}, false, NoMatch},
		{"valueLonger_mismatch", "val", nil, []string{"head", "value", "tail"}, false, NoMatch},
		{"withAsterisk", "v*o", nil, []string{"value", "vulkano", "tail"}, false, Exclude},
		{"withQuestionMark", "vul?ano", nil, []string{"value", "vulkano", "tail"}, false, Exclude},
		{"magicChars", "v[ou]l[kc]ano", nil, []string{"value", "volcano"}, false, Exclude},
		{"wrongPattern_mismatch", "v[ou]l[", nil, []string{"value", "vol["}, false, NoMatch},
		{"glob_fromRootWithSlash", "/value/vul?ano", nil, []string{"value", "vulkano", "tail"}, false, Exclude},
		{"glob_withDomain", "middle/tail/", []string{"value", "volcano"}, []string{"value", "volcano", "middle", "tail"}, true, Exclude},
		{"glob_onlyMatchInDomain_mismatch", "volcano/tail", []string{"value", "volcano"}, []string{"value", "volcano", "tail"}, false, NoMatch},
		{"glob_fromRootWithoutSlash", "value/vul?ano", nil, []string{"value", "vulkano", "tail"}, false, Exclude},
		{"glob_fromRoot_mismatch", "value/vulkano", nil, []string{"value", "volcano"}, false, NoMatch},
		{"glob_fromRoot_tooShort_mismatch", "value/vul?ano", nil, []string{"value"}, false, NoMatch},
		{"glob_fromRoot_notAtRoot_mismatch", "/value/volcano", nil, []string{"value", "value", "volcano"}, false, NoMatch},
		{"glob_leadingAsterisks_atStart", "**/*lue/vol?ano", nil, []string{"value", "volcano", "tail"}, false, Exclude},
		{"glob_leadingAsterisks_notAtStart", "**/*lue/vol?ano", nil, []string{"head", "value", "volcano", "tail"}, false, Exclude},
		{"glob_leadingAsterisks_mismatch", "**/*lue/vol?ano", nil, []string{"head", "value", "Volcano", "tail"}, false, NoMatch},
		{"glob_leadingAsterisks_isDir", "**/*lue/vol?ano/", nil, []string{"head", "value", "volcano", "tail"}, false, Exclude},
		{"glob_leadingAsterisks_isDirAtEnd", "**/*lue/vol?ano/", nil, []string{"head", "value", "volcano"}, true, Exclude},
		{"glob_leadingAsterisks_isDir_mismatch", "**/*lue/vol?ano/", nil, []string{"head", "value", "Colcano"}, true, NoMatch},
		{"glob_leadingAsterisks_isDirNoDirAtEnd_mismatch", "**/*lue/vol?ano/", nil, []string{"head", "value", "volcano"}, false, NoMatch},
		{"glob_tailingAsterisks", "/*lue/vol?ano/**", nil, []string{"value", "volcano", "tail", "moretail"}, false, Exclude},
		{"glob_tailingAsterisks_exactMatch", "/*lue/vol?ano/**", nil, []string{"value", "volcano"}, false, Exclude},
		{"glob_middleAsterisks_emptyMatch", "/*lue/**/vol?ano", nil, []string{"value", "volcano"}, false, Exclude},
		{"glob_middleAsterisks_oneMatch", "/*lue/**/vol?ano", nil, []string{"value", "middle", "volcano"}, false, Exclude},
		{"glob_middleAsterisks_multiMatch", "/*lue/**/vol?ano", nil, []string{"value", "middle1", "middle2", "volcano"}, false, Exclude},
		{"glob_middleAsterisks_isDir_trailing", "/*lue/**/vol?ano/", nil, []string{"value", "middle1", "middle2", "volcano"}, true, Exclude},
		{"glob_middleAsterisks_isDir_trailing_mismatch", "/*lue/**/vol?ano/", nil, []string{"value", "middle1", "middle2", "volcano"}, false, NoMatch},
		{"glob_middleAsterisks_isDir", "/*lue/**/vol?ano/", nil, []string{"value", "middle1", "middle2", "volcano", "tail"}, false, Exclude},
		{"glob_wrongDoubleAsterisk_mismatch", "/*lue/**foo/vol?ano", nil, []string{"value", "foo", "volcano", "tail"}, false, NoMatch},
		{"glob_magicChars", "**/head/v[ou]l[kc]ano", nil, []string{"value", "head", "volcano"}, false, Exclude},
		{"glob_wrongPattern_noTraversal_mismatch", "**/head/v[ou]l[", nil, []string{"value", "head", "vol["}, false, NoMatch},
		{"glob_wrongPattern_onTraversal_mismatch", "/value/**/v[ou]l[", nil, []string{"value", "head", "vol["}, false, NoMatch},
		{"glob_issue_923", "**/android/**/GeneratedPluginRegistrant.java", nil, []string{"packages", "flutter_tools", "lib", "src", "android", "gradle.dart"}, false, NoMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := ParsePattern(tt.pattern, tt.domain)
			if got := p.Match(tt.path, tt.isDir); got != tt.want {
				t.Errorf("ParsePattern(%q, %v).Match(%v, %t) = %v, want %v",
					tt.pattern, tt.domain, tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}
