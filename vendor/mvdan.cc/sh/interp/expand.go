// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"mvdan.cc/sh/syntax"
)

func (r *Runner) expandFormat(format string, args []string) (int, string, error) {
	buf := r.strBuilder()
	esc := false
	var fmts []rune
	initialArgs := len(args)

	for _, c := range format {
		switch {
		case esc:
			esc = false
			switch c {
			case 'n':
				buf.WriteRune('\n')
			case 'r':
				buf.WriteRune('\r')
			case 't':
				buf.WriteRune('\t')
			case '\\':
				buf.WriteRune('\\')
			default:
				buf.WriteRune('\\')
				buf.WriteRune(c)
			}

		case len(fmts) > 0:
			switch c {
			case '%':
				buf.WriteByte('%')
				fmts = nil
			case 'c':
				var b byte
				if len(args) > 0 {
					arg := ""
					arg, args = args[0], args[1:]
					if len(arg) > 0 {
						b = arg[0]
					}
				}
				buf.WriteByte(b)
				fmts = nil
			case '+', '-', ' ':
				if len(fmts) > 1 {
					return 0, "", fmt.Errorf("invalid format char: %c", c)
				}
				fmts = append(fmts, c)
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				fmts = append(fmts, c)
			case 's', 'd', 'i', 'u', 'o', 'x':
				arg := ""
				if len(args) > 0 {
					arg, args = args[0], args[1:]
				}
				var farg interface{} = arg
				if c != 's' {
					n, _ := strconv.ParseInt(arg, 0, 0)
					if c == 'i' || c == 'd' {
						farg = int(n)
					} else {
						farg = uint(n)
					}
					if c == 'i' || c == 'u' {
						c = 'd'
					}
				}
				fmts = append(fmts, c)
				fmt.Fprintf(buf, string(fmts), farg)
				fmts = nil
			default:
				return 0, "", fmt.Errorf("invalid format char: %c", c)
			}
		case c == '\\':
			esc = true
		case args != nil && c == '%':
			// if args == nil, we are not doing format
			// arguments
			fmts = []rune{c}
		default:
			buf.WriteRune(c)
		}
	}
	if len(fmts) > 0 {
		return 0, "", fmt.Errorf("missing format char")
	}
	return initialArgs - len(args), buf.String(), nil
}

func (r *Runner) fieldJoin(parts []fieldPart) string {
	switch len(parts) {
	case 0:
		return ""
	case 1: // short-cut without a string copy
		return parts[0].val
	}
	buf := r.strBuilder()
	for _, part := range parts {
		buf.WriteString(part.val)
	}
	return buf.String()
}

func (r *Runner) escapedGlobField(parts []fieldPart) (escaped string, glob bool) {
	buf := r.strBuilder()
	for _, part := range parts {
		quoted := syntax.QuotePattern(part.val)
		if quoted != part.val {
			if part.quote > quoteNone {
				buf.WriteString(quoted)
			} else {
				buf.WriteString(part.val)
				glob = true
			}
		}
	}
	if glob { // only copy the string if it will be used
		escaped = buf.String()
	}
	return escaped, glob
}

func (r *Runner) Fields(words ...*syntax.Word) []string {
	fields := make([]string, 0, len(words))
	baseDir := syntax.QuotePattern(r.Dir)
	for _, word := range words {
		for _, expWord := range syntax.ExpandBraces(word) {
			for _, field := range r.wordFields(expWord.Parts) {
				path, doGlob := r.escapedGlobField(field)
				var matches []string
				abs := filepath.IsAbs(path)
				if doGlob && !r.opts[optNoGlob] {
					if !abs {
						path = filepath.Join(baseDir, path)
					}
					matches = glob(path, r.opts[optGlobStar])
				}
				if len(matches) == 0 {
					fields = append(fields, r.fieldJoin(field))
					continue
				}
				for _, match := range matches {
					if !abs {
						endSeparator := strings.HasSuffix(match, string(filepath.Separator))
						match, _ = filepath.Rel(r.Dir, match)
						if endSeparator {
							match += string(filepath.Separator)
						}
					}
					fields = append(fields, match)
				}
			}
		}
	}
	return fields
}

func (r *Runner) loneWord(word *syntax.Word) string {
	if word == nil {
		return ""
	}
	field := r.wordField(word.Parts, quoteDouble)
	return r.fieldJoin(field)
}

func (r *Runner) lonePattern(word *syntax.Word) string {
	field := r.wordField(word.Parts, quoteSingle)
	buf := r.strBuilder()
	for _, part := range field {
		if part.quote > quoteNone {
			buf.WriteString(syntax.QuotePattern(part.val))
		} else {
			buf.WriteString(part.val)
		}
	}
	return buf.String()
}

func (r *Runner) expandAssigns(as *syntax.Assign) []*syntax.Assign {
	// Convert "declare $x" into "declare value".
	// Don't use syntax.Parser here, as we only want the basic
	// splitting by '='.
	if as.Name != nil {
		return []*syntax.Assign{as} // nothing to do
	}
	var asgns []*syntax.Assign
	for _, field := range r.Fields(as.Value) {
		as := &syntax.Assign{}
		parts := strings.SplitN(field, "=", 2)
		as.Name = &syntax.Lit{Value: parts[0]}
		if len(parts) == 1 {
			as.Naked = true
		} else {
			as.Value = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.Lit{Value: parts[1]},
			}}
		}
		asgns = append(asgns, as)
	}
	return asgns
}

type fieldPart struct {
	val   string
	quote quoteLevel
}

type quoteLevel uint

const (
	quoteNone quoteLevel = iota
	quoteDouble
	quoteSingle
)

func (r *Runner) wordField(wps []syntax.WordPart, ql quoteLevel) []fieldPart {
	var field []fieldPart
	for i, wp := range wps {
		switch x := wp.(type) {
		case *syntax.Lit:
			s := x.Value
			if i == 0 {
				s = r.expandUser(s)
			}
			if ql == quoteDouble && strings.Contains(s, "\\") {
				buf := r.strBuilder()
				for i := 0; i < len(s); i++ {
					b := s[i]
					if b == '\\' && i+1 < len(s) {
						switch s[i+1] {
						case '\n': // remove \\\n
							i++
							continue
						case '\\', '$', '`': // special chars
							continue
						}
					}
					buf.WriteByte(b)
				}
				s = buf.String()
			}
			field = append(field, fieldPart{val: s})
		case *syntax.SglQuoted:
			fp := fieldPart{quote: quoteSingle, val: x.Value}
			if x.Dollar {
				_, fp.val, _ = r.expandFormat(fp.val, nil)
			}
			field = append(field, fp)
		case *syntax.DblQuoted:
			for _, part := range r.wordField(x.Parts, quoteDouble) {
				part.quote = quoteDouble
				field = append(field, part)
			}
		case *syntax.ParamExp:
			field = append(field, fieldPart{val: r.paramExp(x)})
		case *syntax.CmdSubst:
			field = append(field, fieldPart{val: r.cmdSubst(x)})
		case *syntax.ArithmExp:
			field = append(field, fieldPart{
				val: strconv.Itoa(r.arithm(x.X)),
			})
		default:
			panic(fmt.Sprintf("unhandled word part: %T", x))
		}
	}
	return field
}

func (r *Runner) cmdSubst(cs *syntax.CmdSubst) string {
	r2 := r.sub()
	buf := r.strBuilder()
	r2.Stdout = buf
	r2.stmts(cs.StmtList)
	r.setErr(r2.err)
	return strings.TrimRight(buf.String(), "\n")
}

func (r *Runner) wordFields(wps []syntax.WordPart) [][]fieldPart {
	fields := r.fieldsAlloc[:0]
	curField := r.fieldAlloc[:0]
	allowEmpty := false
	flush := func() {
		if len(curField) == 0 {
			return
		}
		fields = append(fields, curField)
		curField = nil
	}
	splitAdd := func(val string) {
		for i, field := range strings.FieldsFunc(val, r.ifsRune) {
			if i > 0 {
				flush()
			}
			curField = append(curField, fieldPart{val: field})
		}
	}
	for i, wp := range wps {
		switch x := wp.(type) {
		case *syntax.Lit:
			s := x.Value
			if i == 0 {
				s = r.expandUser(s)
			}
			if strings.Contains(s, "\\") {
				buf := r.strBuilder()
				for i := 0; i < len(s); i++ {
					b := s[i]
					if b == '\\' {
						i++
						b = s[i]
					}
					buf.WriteByte(b)
				}
				s = buf.String()
			}
			curField = append(curField, fieldPart{val: s})
		case *syntax.SglQuoted:
			allowEmpty = true
			fp := fieldPart{quote: quoteSingle, val: x.Value}
			if x.Dollar {
				_, fp.val, _ = r.expandFormat(fp.val, nil)
			}
			curField = append(curField, fp)
		case *syntax.DblQuoted:
			allowEmpty = true
			if len(x.Parts) == 1 {
				pe, _ := x.Parts[0].(*syntax.ParamExp)
				if elems := r.quotedElems(pe); elems != nil {
					for i, elem := range elems {
						if i > 0 {
							flush()
						}
						curField = append(curField, fieldPart{
							quote: quoteDouble,
							val:   elem,
						})
					}
					continue
				}
			}
			for _, part := range r.wordField(x.Parts, quoteDouble) {
				part.quote = quoteDouble
				curField = append(curField, part)
			}
		case *syntax.ParamExp:
			splitAdd(r.paramExp(x))
		case *syntax.CmdSubst:
			splitAdd(r.cmdSubst(x))
		case *syntax.ArithmExp:
			curField = append(curField, fieldPart{
				val: strconv.Itoa(r.arithm(x.X)),
			})
		default:
			panic(fmt.Sprintf("unhandled word part: %T", x))
		}
	}
	flush()
	if allowEmpty && len(fields) == 0 {
		fields = append(fields, curField)
	}
	return fields
}

func (r *Runner) expandUser(field string) string {
	if len(field) == 0 || field[0] != '~' {
		return field
	}
	name := field[1:]
	rest := ""
	if i := strings.Index(name, "/"); i >= 0 {
		rest = name[i:]
		name = name[:i]
	}
	if name == "" {
		return r.getVar("HOME") + rest
	}
	u, err := user.Lookup(name)
	if err != nil {
		return field
	}
	return u.HomeDir + rest
}

func match(pattern, name string) bool {
	expr, err := syntax.TranslatePattern(pattern, true)
	if err != nil {
		return false
	}
	rx := regexp.MustCompile("^" + expr + "$")
	return rx.MatchString(name)
}

func findAllIndex(pattern, name string, n int) [][]int {
	expr, err := syntax.TranslatePattern(pattern, true)
	if err != nil {
		return nil
	}
	rx := regexp.MustCompile(expr)
	return rx.FindAllStringIndex(name, n)
}

func hasGlob(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}

var rxGlobStar = regexp.MustCompile(".*")

func glob(pattern string, globStar bool) []string {
	parts := strings.Split(pattern, string(filepath.Separator))
	matches := []string{"."}
	if filepath.IsAbs(pattern) {
		if parts[0] == "" {
			// unix-like
			matches[0] = string(filepath.Separator)
		} else {
			// windows (for some reason it won't work without the
			// trailing separator)
			matches[0] = parts[0] + string(filepath.Separator)
		}
		parts = parts[1:]
	}
	for _, part := range parts {
		if part == "**" && globStar {
			for i := range matches {
				// "a/**" should match "a/ a/b a/b/c ..."; note
				// how the zero-match case has a trailing
				// separator.
				matches[i] += string(filepath.Separator)
			}
			// expand all the possible levels of **
			latest := matches
			for {
				var newMatches []string
				for _, dir := range latest {
					newMatches = globDir(dir, rxGlobStar, newMatches)
				}
				if len(newMatches) == 0 {
					// not another level of directories to
					// try; stop
					break
				}
				matches = append(matches, newMatches...)
				latest = newMatches
			}
			continue
		}
		expr, err := syntax.TranslatePattern(part, true)
		if err != nil {
			return nil
		}
		rx := regexp.MustCompile("^" + expr + "$")
		var newMatches []string
		for _, dir := range matches {
			newMatches = globDir(dir, rx, newMatches)
		}
		matches = newMatches
	}
	return matches
}

func globDir(dir string, rx *regexp.Regexp, matches []string) []string {
	d, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer d.Close()

	names, _ := d.Readdirnames(-1)
	sort.Strings(names)

	for _, name := range names {
		if !strings.HasPrefix(rx.String(), `^\.`) && name[0] == '.' {
			continue
		}
		if rx.MatchString(name) {
			matches = append(matches, filepath.Join(dir, name))
		}
	}
	return matches
}
