package templater

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sebdah/goldie/v2"

	"github.com/go-task/task/v3/taskfile/ast"
)

// The two tests in this file snapshot the entire template surface: the set of
// function names, and the rendered output of a representative expression per
// function. They exist to make any change of templating library auditable —
// the golden diff is the exhaustive list of what changed for users.
//
// Expressions must stay deterministic and platform independent, so anything
// depending on the clock, the filesystem, the environment or randomness is
// deliberately absent (`now`, `uuid`, `randInt`, `env`, `os*`, `exeExt`, `OS`,
// `ARCH`, `spew`, `getHostByName`). Those are covered by unit tests instead.

// TestMain pins the local timezone so that the date expressions below render
// identically on every machine. A consequence is that this golden cannot show
// the local-vs-UTC difference between sprig's `date`/`toDate` and sprout's —
// that one belongs in the migration documentation.
func TestMain(m *testing.M) {
	time.Local = time.UTC
	os.Exit(m.Run())
}

func TestFuncNames(t *testing.T) {
	t.Parallel()

	names := slices.Sorted(maps.Keys(templateFuncs))

	g := goldie.New(t)
	g.Assert(t, "func_names", []byte(strings.Join(names, "\n")+"\n"))
}

func TestFuncBehaviour(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	for _, group := range funcBehaviourGroups {
		fmt.Fprintf(&b, "## %s\n\n", group.name)
		for _, expr := range group.exprs {
			cache := &Cache{Vars: ast.NewVars()}
			got := ReplaceWithExtra(expr, cache, nil)
			fmt.Fprintf(&b, "%s\n", expr)
			if err := cache.Err(); err != nil {
				fmt.Fprintf(&b, "\t! %s\n", normalizeTemplateError(err))
			} else {
				fmt.Fprintf(&b, "\t= %s\n", strings.ReplaceAll(got, "\n", "\\n"))
			}
		}
		b.WriteString("\n")
	}

	g := goldie.New(t)
	g.Assert(t, "func_behaviour", []byte(b.String()))
}

// normalizeTemplateError strips the position prefix that text/template adds to
// execution errors, which is noise in the golden and shifts whenever an
// expression is added to a group.
func normalizeTemplateError(err error) string {
	s := err.Error()
	if _, after, found := strings.Cut(s, `at <`); found {
		return "at <" + after
	}
	return s
}

var funcBehaviourGroups = []struct {
	name  string
	exprs []string
}{
	{
		// The ten functions whose argument order differs between sprig and
		// sprout. Written here in sprig order, which is what users' Taskfiles
		// contain today.
		"argument order — sprig order (target first)",
		[]string{
			`{{ get (dict "a" "b") "a" }}`,
			`{{ set (dict "a" "b") "c" "d" | toJson }}`,
			`{{ unset (dict "a" "b" "c" "d") "a" | toJson }}`,
			`{{ hasKey (dict "a" "b") "a" }}`,
			`{{ pick (dict "a" "1" "b" "2") "a" | toJson }}`,
			`{{ omit (dict "a" "1" "b" "2") "a" | toJson }}`,
			`{{ append (list 1 2) 3 | toJson }}`,
			`{{ push (list 1 2) 3 | toJson }}`,
			`{{ prepend (list 2 3) 1 | toJson }}`,
			`{{ without (list 1 2 3) 2 | toJson }}`,
			`{{ slice (list 1 2 3 4) 1 3 | toJson }}`,
		},
	},
	{
		// Same ten functions in sprout order. Today these mostly fail; after
		// the migration both forms must work.
		"argument order — sprout order (target last)",
		[]string{
			`{{ dict "a" "b" | get "a" }}`,
			`{{ dict "a" "b" | set "c" "d" | toJson }}`,
			`{{ dict "a" "b" "c" "d" | unset "a" | toJson }}`,
			`{{ dict "a" "b" | hasKey "a" }}`,
			`{{ list 1 2 | append 3 | toJson }}`,
			`{{ list 2 3 | prepend 1 | toJson }}`,
			`{{ dict "a" "1" "b" "2" | pick "a" | toJson }}`,
			`{{ dict "a" "1" "b" "2" | omit "a" | toJson }}`,
			`{{ list 1 2 3 | without 2 | toJson }}`,
			`{{ list 1 2 3 4 | slice 1 3 | toJson }}`,
		},
	},
	{
		"maps — unchanged signatures",
		[]string{
			`{{ dict "a" 1 "b" 2 | toJson }}`,
			`{{ keys (dict "b" 1 "a" 2) | sortAlpha | toJson }}`,
			`{{ values (dict "a" 1) | toJson }}`,
			`{{ pluck "a" (dict "a" 1) (dict "a" 2) | toJson }}`,
			`{{ dig "a" "b" "fallback" (dict "a" (dict "b" "found")) }}`,
			`{{ dig "a" "missing" "fallback" (dict "a" (dict "b" "found")) }}`,
			`{{ dig "a.b" "fallback" (dict "a" (dict "b" "found")) }}`,
			`{{ merge (dict "a" 1) (dict "b" 2) | toJson }}`,
			`{{ merge (dict "a" 1) (dict "a" 0) | toJson }}`,
		},
	},
	{
		"lists",
		[]string{
			`{{ list 1 2 3 | toJson }}`,
			`{{ tuple 1 2 3 | toJson }}`,
			`{{ first (list 1 2 3) }}`,
			`{{ last (list 1 2 3) }}`,
			`{{ rest (list 1 2 3) | toJson }}`,
			`{{ initial (list 1 2 3) | toJson }}`,
			`{{ reverse (list 1 2 3) | toJson }}`,
			`{{ uniq (list 1 1 2) | toJson }}`,
			`{{ compact (list 1 "" 2) | toJson }}`,
			`{{ concat (list 1) (list 2) | toJson }}`,
			`{{ chunk 2 (list 1 2 3) | toJson }}`,
			`{{ has 2 (list 1 2 3) }}`,
			`{{ sortAlpha (list "b" "a") | toJson }}`,
			`{{ splitList "," "a,b,c" | toJson }}`,
			`{{ toStrings (list 1 2) | toJson }}`,
			`{{ until 3 | toJson }}`,
			`{{ untilStep 0 6 2 | toJson }}`,
			`{{ seq 1 3 }}`,
			`{{ join "," (list "a" "b") }}`,
		},
	},
	{
		"strings",
		[]string{
			`{{ trim "  x  " }}`,
			`{{ trimAll "-" "-x-" }}`,
			`{{ trimall "-" "-x-" }}`,
			`{{ trimPrefix "a" "ab" }}`,
			`{{ trimSuffix "b" "ab" }}`,
			`{{ upper "abc" }}`,
			`{{ lower "ABC" }}`,
			`{{ title "hello world" }}`,
			`{{ title "hello wORLD" }}`,
			`{{ trunc 3 "foobar" }}`,
			`{{ trunc -3 "foobar" }}`,
			`{{ substr 0 3 "foobar" }}`,
			`{{ substr 0 -3 "foobar" }}`,
			`{{ repeat 3 "x" }}`,
			`{{ contains "oo" "foobar" }}`,
			`{{ hasPrefix "foo" "foobar" }}`,
			`{{ hasSuffix "bar" "foobar" }}`,
			`{{ quote "x" }}`,
			`{{ squote "x" }}`,
			`{{ cat "a" "b" }}`,
			`{{ indent 2 "x" }}`,
			`{{ nindent 2 "x" }}`,
			`{{ replace "a" "b" "aa" }}`,
			`{{ plural "one" "many" 2 }}`,
			`{{ split "," "a,b" | toJson }}`,
			`{{ splitn "," 2 "a,b,c" | toJson }}`,
			`{{ toString 42 }}`,
		},
	},
	{
		"numbers",
		[]string{
			`{{ add 1 2 }}`,
			`{{ add1 1 }}`,
			`{{ sub 5 2 }}`,
			`{{ mul 2 3 }}`,
			`{{ div 6 2 }}`,
			`{{ mod 5 3 }}`,
			`{{ max 1 5 3 }}`,
			`{{ min 1 5 3 }}`,
			`{{ biggest 1 5 3 }}`,
			`{{ maxf 1.5 2.5 }}`,
			`{{ minf 1.5 2.5 }}`,
			`{{ ceil 1.1 }}`,
			`{{ floor 1.9 }}`,
			`{{ round 1.55 1 }}`,
			`{{ atoi "42" }}`,
			`{{ atoi "abc" }}`,
			`{{ int "42" }}`,
			`{{ int64 "42" }}`,
			`{{ float64 "1.5" }}`,
			`{{ toDecimal "0777" }}`,
		},
	},
	{
		"defaults and flow",
		[]string{
			`{{ default "d" "" }}`,
			`{{ default "d" "x" }}`,
			`{{ empty "" }}`,
			`{{ empty 0 }}`,
			`{{ coalesce "" "x" }}`,
			`{{ all 1 1 }}`,
			`{{ any 0 1 }}`,
			`{{ ternary "y" "n" true }}`,
			`{{ fail "boom" }}`,
		},
	},
	{
		"encoding",
		[]string{
			`{{ toJson (dict "a" 1) }}`,
			`{{ toPrettyJson (dict "a" 1) }}`,
			`{{ toRawJson (dict "a" "<b>") }}`,
			`{{ fromJson "{\"a\":1}" | toJson }}`,
			`{{ fromJson "not json" | toJson }}`,
			`{{ mustFromJson "not json" | toJson }}`,
			`{{ toYaml (dict "a" 1) }}`,
			`{{ fromYaml "a: 1" | toJson }}`,
			`{{ mustFromYaml "a: :" | toJson }}`,
			`{{ b64enc "hello" }}`,
			`{{ b64dec "aGVsbG8=" }}`,
			`{{ b32enc "hello" }}`,
			`{{ b32dec "NBSWY3DP" }}`,
		},
	},
	{
		"regex",
		[]string{
			`{{ regexMatch "^a" "abc" }}`,
			`{{ regexFind "[0-9]+" "abc123" }}`,
			`{{ regexFindAll "[0-9]" "a1b2" -1 | toJson }}`,
			`{{ regexReplaceAll "[0-9]" "abc123" "#" }}`,
			`{{ regexReplaceAllLiteral "[0-9]" "abc123" "#" }}`,
			`{{ regexSplit "," "a,b" -1 | toJson }}`,
			`{{ regexQuoteMeta "a.b" }}`,
			`{{ mustRegexFind "[" "abc" }}`,
		},
	},
	{
		"reflection",
		[]string{
			`{{ typeOf 1 }}`,
			`{{ typeIs "int" 1 }}`,
			`{{ typeIsLike "int" 1 }}`,
			`{{ kindOf 1 }}`,
			`{{ kindOf (list 1) }}`,
			`{{ kindIs "int" 1 }}`,
			`{{ deepEqual (list 1) (list 1) }}`,
		},
	},
	{
		"checksums",
		[]string{
			`{{ sha1sum "x" }}`,
			`{{ sha256sum "x" }}`,
			`{{ adler32sum "x" }}`,
		},
	},
	{
		// path.* semantics — slash based, therefore identical on every platform.
		"paths",
		[]string{
			`{{ base "/foo/bar.txt" }}`,
			`{{ dir "/foo/bar.txt" }}`,
			`{{ ext "/foo/bar.txt" }}`,
			`{{ clean "/foo//bar" }}`,
			`{{ isAbs "/foo" }}`,
		},
	},
	{
		"dates — fixed epoch, explicit zone",
		[]string{
			`{{ dateInZone "2006-01-02T15:04:05" 0 "UTC" }}`,
			`{{ date "2006-01-02T15:04:05" 0 }}`,
			`{{ dateModify "1h" (toDate "2006-01-02T15:04:05Z07:00" "2020-01-01T00:00:00Z") }}`,
			`{{ date_modify "1h" (toDate "2006-01-02T15:04:05Z07:00" "2020-01-01T00:00:00Z") }}`,
			`{{ toDate "2006-01-02" "2020-01-01" }}`,
			`{{ unixEpoch (toDate "2006-01-02" "2020-01-01") }}`,
			`{{ duration 90 }}`,
			`{{ durationRound "1h35m30s" }}`,
			`{{ htmlDate 0 }}`,
		},
	},
	{
		"urls",
		[]string{
			`{{ urlParse "http://example.com/a?b=c" | toJson }}`,
			`{{ urlJoin (dict "scheme" "http" "host" "example.com" "path" "/a") }}`,
		},
	},
	{
		"task's own functions",
		[]string{
			`{{ numCPU | kindOf }}`,
			`{{ catLines "a\nb" }}`,
			`{{ splitLines "a\nb" | toJson }}`,
			`{{ toSlash "a/b" }}`,
			`{{ fromSlash "a/b" | kindOf }}`,
			`{{ ToSlash "a/b" }}`,
			`{{ shellQuote "a b" }}`,
			`{{ q "a b" }}`,
			`{{ splitArgs "a b c" | toJson }}`,
			`{{ IsSH }}`,
			`{{ joinUrl "http://localhost" "a" "b" }}`,
			`{{ mustToYaml (dict "a" 1) }}`,
			`{{ randIntN 1 }}`,
		},
	},
}
