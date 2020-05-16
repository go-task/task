// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package expand

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/pattern"
	"mvdan.cc/sh/v3/syntax"
)

// A Config specifies details about how shell expansion should be performed. The
// zero value is a valid configuration.
type Config struct {
	// Env is used to get and set environment variables when performing
	// shell expansions. Some special parameters are also expanded via this
	// interface, such as:
	//
	//   * "#", "@", "*", "0"-"9" for the shell's parameters
	//   * "?", "$", "PPID" for the shell's status and process
	//   * "HOME foo" to retrieve user foo's home directory (if unset,
	//     os/user.Lookup will be used)
	//
	// If nil, there are no environment variables set. Use
	// ListEnviron(os.Environ()...) to use the system's environment
	// variables.
	Env Environ

	// CmdSubst expands a command substitution node, writing its standard
	// output to the provided io.Writer.
	//
	// If nil, encountering a command substitution will result in an
	// UnexpectedCommandError.
	CmdSubst func(io.Writer, *syntax.CmdSubst) error

	// ProcSubst expands a process substitution node.
	//
	// Note that this feature is a work in progress, and the signature of
	// this field might change until #451 is completely fixed.
	ProcSubst func(*syntax.ProcSubst) (string, error)

	// ReadDir is used for file path globbing. If nil, globbing is disabled.
	// Use ioutil.ReadDir to use the filesystem directly.
	ReadDir func(string) ([]os.FileInfo, error)

	// GlobStar corresponds to the shell option that allows globbing with
	// "**".
	GlobStar bool

	bufferAlloc bytes.Buffer
	fieldAlloc  [4]fieldPart
	fieldsAlloc [4][]fieldPart

	ifs string
	// A pointer to a parameter expansion node, if we're inside one.
	// Necessary for ${LINENO}.
	curParam *syntax.ParamExp
}

// UnexpectedCommandError is returned if a command substitution is encountered
// when Config.CmdSubst is nil.
type UnexpectedCommandError struct {
	Node *syntax.CmdSubst
}

func (u UnexpectedCommandError) Error() string {
	return fmt.Sprintf("unexpected command substitution at %s", u.Node.Pos())
}

var zeroConfig = &Config{}

func prepareConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = zeroConfig
	}
	if cfg.Env == nil {
		cfg.Env = FuncEnviron(func(string) string { return "" })
	}

	cfg.ifs = " \t\n"
	if vr := cfg.Env.Get("IFS"); vr.IsSet() {
		cfg.ifs = vr.String()
	}
	return cfg
}

func (cfg *Config) ifsRune(r rune) bool {
	for _, r2 := range cfg.ifs {
		if r == r2 {
			return true
		}
	}
	return false
}

func (cfg *Config) ifsJoin(strs []string) string {
	sep := ""
	if cfg.ifs != "" {
		sep = cfg.ifs[:1]
	}
	return strings.Join(strs, sep)
}

func (cfg *Config) strBuilder() *bytes.Buffer {
	b := &cfg.bufferAlloc
	b.Reset()
	return b
}

func (cfg *Config) envGet(name string) string {
	return cfg.Env.Get(name).String()
}

func (cfg *Config) envSet(name, value string) error {
	wenv, ok := cfg.Env.(WriteEnviron)
	if !ok {
		return fmt.Errorf("environment is read-only")
	}
	return wenv.Set(name, Variable{Kind: String, Str: value})
}

// Literal expands a single shell word. It is similar to Fields, but the result
// is a single string. This is the behavior when a word is used as the value in
// a shell variable assignment, for example.
//
// The config specifies shell expansion options; nil behaves the same as an
// empty config.
func Literal(cfg *Config, word *syntax.Word) (string, error) {
	if word == nil {
		return "", nil
	}
	cfg = prepareConfig(cfg)
	field, err := cfg.wordField(word.Parts, quoteNone)
	if err != nil {
		return "", err
	}
	return cfg.fieldJoin(field), nil
}

// Document expands a single shell word as if it were within double quotes. It
// is simlar to Literal, but without brace expansion, tilde expansion, and
// globbing.
//
// The config specifies shell expansion options; nil behaves the same as an
// empty config.
func Document(cfg *Config, word *syntax.Word) (string, error) {
	if word == nil {
		return "", nil
	}
	cfg = prepareConfig(cfg)
	field, err := cfg.wordField(word.Parts, quoteDouble)
	if err != nil {
		return "", err
	}
	return cfg.fieldJoin(field), nil
}

const patMode = pattern.Filenames | pattern.Braces

// Pattern expands a single shell word as a pattern, using syntax.QuotePattern
// on any non-quoted parts of the input word. The result can be used on
// syntax.TranslatePattern directly.
//
// The config specifies shell expansion options; nil behaves the same as an
// empty config.
func Pattern(cfg *Config, word *syntax.Word) (string, error) {
	cfg = prepareConfig(cfg)
	field, err := cfg.wordField(word.Parts, quoteNone)
	if err != nil {
		return "", err
	}
	buf := cfg.strBuilder()
	for _, part := range field {
		if part.quote > quoteNone {
			buf.WriteString(pattern.QuoteMeta(part.val, patMode))
		} else {
			buf.WriteString(part.val)
		}
	}
	return buf.String(), nil
}

// Format expands a format string with a number of arguments, following the
// shell's format specifications. These include printf(1), among others.
//
// The resulting string is returned, along with the number of arguments used.
//
// The config specifies shell expansion options; nil behaves the same as an
// empty config.
func Format(cfg *Config, format string, args []string) (string, int, error) {
	cfg = prepareConfig(cfg)
	buf := cfg.strBuilder()
	var fmts []byte
	initialArgs := len(args)

	for i := 0; i < len(format); i++ {
		// readDigits reads from 0 to max digits, either octal or
		// hexadecimal.
		readDigits := func(max int, hex bool) string {
			j := 0
			for ; j < max; j++ {
				c := format[i+j]
				if (c >= '0' && c <= '9') ||
					(hex && c >= 'a' && c <= 'f') ||
					(hex && c >= 'A' && c <= 'F') {
					// valid octal or hex char
				} else {
					break
				}
			}
			digits := format[i : i+j]
			i += j - 1 // -1 since the outer loop does i++
			return digits
		}
		c := format[i]
		switch {
		case c == '\\': // escaped
			i++
			switch c = format[i]; c {
			case 'a': // bell
				buf.WriteByte('\a')
			case 'b': // backspace
				buf.WriteByte('\b')
			case 'e', 'E': // escape
				buf.WriteByte('\x1b')
			case 'f': // form feed
				buf.WriteByte('\f')
			case 'n': // new line
				buf.WriteByte('\n')
			case 'r': // carriage return
				buf.WriteByte('\r')
			case 't': // horizontal tab
				buf.WriteByte('\t')
			case 'v': // vertical tab
				buf.WriteByte('\v')
			case '\\', '\'', '"', '?': // just the character
				buf.WriteByte(c)
			case '0', '1', '2', '3', '4', '5', '6', '7':
				digits := readDigits(3, false)
				// if digits don't fit in 8 bits, 0xff via strconv
				n, _ := strconv.ParseUint(digits, 8, 8)
				buf.WriteByte(byte(n))
			case 'x', 'u', 'U':
				i++
				max := 2
				if c == 'u' {
					max = 4
				} else if c == 'U' {
					max = 8
				}
				digits := readDigits(max, true)
				if len(digits) > 0 {
					// can't error
					n, _ := strconv.ParseUint(digits, 16, 32)
					if c == 'x' {
						// always as a single byte
						buf.WriteByte(byte(n))
					} else {
						buf.WriteRune(rune(n))
					}
					break
				}
				fallthrough
			default: // no escape sequence
				buf.WriteByte('\\')
				buf.WriteByte(c)
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
					return "", 0, fmt.Errorf("invalid format char: %c", c)
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
				return "", 0, fmt.Errorf("invalid format char: %c", c)
			}
		case args != nil && c == '%':
			// if args == nil, we are not doing format
			// arguments
			fmts = []byte{c}
		default:
			buf.WriteByte(c)
		}
	}
	if len(fmts) > 0 {
		return "", 0, fmt.Errorf("missing format char")
	}
	return buf.String(), initialArgs - len(args), nil
}

func (cfg *Config) fieldJoin(parts []fieldPart) string {
	switch len(parts) {
	case 0:
		return ""
	case 1: // short-cut without a string copy
		return parts[0].val
	}
	buf := cfg.strBuilder()
	for _, part := range parts {
		buf.WriteString(part.val)
	}
	return buf.String()
}

func (cfg *Config) escapedGlobField(parts []fieldPart) (escaped string, glob bool) {
	buf := cfg.strBuilder()
	for _, part := range parts {
		if part.quote > quoteNone {
			buf.WriteString(pattern.QuoteMeta(part.val, patMode))
			continue
		}
		buf.WriteString(part.val)
		if pattern.HasMeta(part.val, patMode) {
			glob = true
		}
	}
	if glob { // only copy the string if it will be used
		escaped = buf.String()
	}
	return escaped, glob
}

// Fields expands a number of words as if they were arguments in a shell
// command. This includes brace expansion, tilde expansion, parameter expansion,
// command substitution, arithmetic expansion, and quote removal.
func Fields(cfg *Config, words ...*syntax.Word) ([]string, error) {
	cfg = prepareConfig(cfg)
	fields := make([]string, 0, len(words))
	dir := cfg.envGet("PWD")
	for _, word := range words {
		word := *word // make a copy, since SplitBraces replaces the Parts slice
		afterBraces := []*syntax.Word{&word}
		if syntax.SplitBraces(&word) {
			afterBraces = Braces(&word)
		}
		for _, word2 := range afterBraces {
			wfields, err := cfg.wordFields(word2.Parts)
			if err != nil {
				return nil, err
			}
			for _, field := range wfields {
				path, doGlob := cfg.escapedGlobField(field)
				var matches []string
				if doGlob && cfg.ReadDir != nil {
					matches, err = cfg.glob(dir, path)
					if err != nil {
						return nil, err
					}
					if len(matches) > 0 {
						fields = append(fields, matches...)
						continue
					}
				}
				fields = append(fields, cfg.fieldJoin(field))
			}
		}
	}
	return fields, nil
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

func (cfg *Config) wordField(wps []syntax.WordPart, ql quoteLevel) ([]fieldPart, error) {
	var field []fieldPart
	for i, wp := range wps {
		switch x := wp.(type) {
		case *syntax.Lit:
			s := x.Value
			if i == 0 && ql == quoteNone {
				if prefix, rest := cfg.expandUser(s); prefix != "" {
					// TODO: return two separate fieldParts,
					// like in wordFields?
					s = prefix + rest
				}
			}
			if ql == quoteDouble && strings.Contains(s, "\\") {
				buf := cfg.strBuilder()
				for i := 0; i < len(s); i++ {
					b := s[i]
					if b == '\\' && i+1 < len(s) {
						switch s[i+1] {
						case '"', '\\', '$', '`': // special chars
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
				fp.val, _, _ = Format(cfg, fp.val, nil)
			}
			field = append(field, fp)
		case *syntax.DblQuoted:
			wfield, err := cfg.wordField(x.Parts, quoteDouble)
			if err != nil {
				return nil, err
			}
			for _, part := range wfield {
				part.quote = quoteDouble
				field = append(field, part)
			}
		case *syntax.ParamExp:
			val, err := cfg.paramExp(x)
			if err != nil {
				return nil, err
			}
			field = append(field, fieldPart{val: val})
		case *syntax.CmdSubst:
			val, err := cfg.cmdSubst(x)
			if err != nil {
				return nil, err
			}
			field = append(field, fieldPart{val: val})
		case *syntax.ArithmExp:
			n, err := Arithm(cfg, x.X)
			if err != nil {
				return nil, err
			}
			field = append(field, fieldPart{val: strconv.Itoa(n)})
		case *syntax.ProcSubst:
			path, err := cfg.ProcSubst(x)
			if err != nil {
				return nil, err
			}
			field = append(field, fieldPart{val: path})
		default:
			panic(fmt.Sprintf("unhandled word part: %T", x))
		}
	}
	return field, nil
}

func (cfg *Config) cmdSubst(cs *syntax.CmdSubst) (string, error) {
	if cfg.CmdSubst == nil {
		return "", UnexpectedCommandError{Node: cs}
	}
	buf := cfg.strBuilder()
	if err := cfg.CmdSubst(buf, cs); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func (cfg *Config) wordFields(wps []syntax.WordPart) ([][]fieldPart, error) {
	fields := cfg.fieldsAlloc[:0]
	curField := cfg.fieldAlloc[:0]
	allowEmpty := false
	flush := func() {
		if len(curField) == 0 {
			return
		}
		fields = append(fields, curField)
		curField = nil
	}
	splitAdd := func(val string) {
		for i, field := range strings.FieldsFunc(val, cfg.ifsRune) {
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
				prefix, rest := cfg.expandUser(s)
				curField = append(curField, fieldPart{
					quote: quoteSingle,
					val:   prefix,
				})
				s = rest
			}
			if strings.Contains(s, "\\") {
				buf := cfg.strBuilder()
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
				fp.val, _, _ = Format(cfg, fp.val, nil)
			}
			curField = append(curField, fp)
		case *syntax.DblQuoted:
			if len(x.Parts) == 1 {
				pe, _ := x.Parts[0].(*syntax.ParamExp)
				if elems := cfg.quotedElemFields(pe); elems != nil {
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
			allowEmpty = true
			wfield, err := cfg.wordField(x.Parts, quoteDouble)
			if err != nil {
				return nil, err
			}
			for _, part := range wfield {
				part.quote = quoteDouble
				curField = append(curField, part)
			}
		case *syntax.ParamExp:
			val, err := cfg.paramExp(x)
			if err != nil {
				return nil, err
			}
			splitAdd(val)
		case *syntax.CmdSubst:
			val, err := cfg.cmdSubst(x)
			if err != nil {
				return nil, err
			}
			splitAdd(val)
		case *syntax.ArithmExp:
			n, err := Arithm(cfg, x.X)
			if err != nil {
				return nil, err
			}
			curField = append(curField, fieldPart{val: strconv.Itoa(n)})
		case *syntax.ProcSubst:
			path, err := cfg.ProcSubst(x)
			if err != nil {
				return nil, err
			}
			splitAdd(path)
		default:
			panic(fmt.Sprintf("unhandled word part: %T", x))
		}
	}
	flush()
	if allowEmpty && len(fields) == 0 {
		fields = append(fields, curField)
	}
	return fields, nil
}

// quotedElemFields returns the list of elements resulting from a quoted
// parameter expansion if it was in the form of ${*}, ${@}, ${foo[*], ${foo[@]},
// or ${!foo@}.
func (cfg *Config) quotedElemFields(pe *syntax.ParamExp) []string {
	if pe == nil || pe.Length || pe.Width {
		return nil
	}
	if pe.Excl {
		if pe.Names == syntax.NamesPrefixWords {
			return cfg.namesByPrefix(pe.Param.Value)
		}
		return nil
	}
	name := pe.Param.Value
	switch name {
	case "*":
		return []string{cfg.ifsJoin(cfg.Env.Get(name).List)}
	case "@":
		return cfg.Env.Get(name).List
	}
	switch nodeLit(pe.Index) {
	case "@":
		if vr := cfg.Env.Get(name); vr.Kind == Indexed {
			return vr.List
		}
	case "*":
		if vr := cfg.Env.Get(name); vr.Kind == Indexed {
			return []string{cfg.ifsJoin(vr.List)}
		}
	}
	return nil
}

func (cfg *Config) expandUser(field string) (prefix, rest string) {
	if len(field) == 0 || field[0] != '~' {
		return "", field
	}
	name := field[1:]
	if i := strings.Index(name, "/"); i >= 0 {
		rest = name[i:]
		name = name[:i]
	}
	if name == "" {
		// Current user; try via "HOME", otherwise fall back to the
		// system's appropriate home dir env var. Don't use os/user, as
		// that's overkill. We can't use os.UserHomeDir, because we want
		// to use cfg.Env, and we always want to check "HOME" first.

		if vr := cfg.Env.Get("HOME"); vr.IsSet() {
			return vr.String(), rest
		}

		if runtime.GOOS == "windows" {
			if vr := cfg.Env.Get("USERPROFILE"); vr.IsSet() {
				return vr.String(), rest
			}
		}
		return "", field
	}

	// Not the current user; try via "HOME <name>", otherwise fall back to
	// os/user. There isn't a way to lookup user home dirs without cgo.

	if vr := cfg.Env.Get("HOME " + name); vr.IsSet() {
		return vr.String(), rest
	}

	u, err := user.Lookup(name)
	if err != nil {
		return "", field
	}
	return u.HomeDir, rest
}

func findAllIndex(pat, name string, n int) [][]int {
	expr, err := pattern.Regexp(pat, 0)
	if err != nil {
		return nil
	}
	rx := regexp.MustCompile(expr)
	return rx.FindAllStringIndex(name, n)
}

var rxGlobStar = regexp.MustCompile(".*")

// pathJoin2 is a simpler version of filepath.Join without cleaning the result,
// since that's needed for globbing.
func pathJoin2(elem1, elem2 string) string {
	if elem1 == "" {
		return elem2
	}
	if strings.HasSuffix(elem1, string(filepath.Separator)) {
		return elem1 + elem2
	}
	return elem1 + string(filepath.Separator) + elem2
}

// pathSplit splits a file path into its elements, retaining empty ones. Before
// splitting, slashes are replaced with filepath.Separator, so that splitting
// Unix paths on Windows works as well.
func pathSplit(path string) []string {
	path = filepath.FromSlash(path)
	return strings.Split(path, string(filepath.Separator))
}

func (cfg *Config) glob(base, pat string) ([]string, error) {
	parts := pathSplit(pat)
	matches := []string{""}
	if filepath.IsAbs(pat) {
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
	for i, part := range parts {
		wantDir := i < len(parts)-1
		switch {
		case part == "", part == ".", part == "..":
			for i, dir := range matches {
				matches[i] = pathJoin2(dir, part)
			}
			continue
		case part == "**" && cfg.GlobStar:
			for i, match := range matches {
				// "a/**" should match "a/ a/b a/b/cfg ..."; note
				// how the zero-match case has a trailing
				// separator.
				matches[i] = pathJoin2(match, "")
			}
			// expand all the possible levels of **
			latest := matches
			for {
				var newMatches []string
				for _, dir := range latest {
					var err error
					newMatches, err = cfg.globDir(base, dir, rxGlobStar, wantDir, newMatches)
					if err != nil {
						return nil, err
					}
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
		expr, err := pattern.Regexp(part, pattern.Filenames)
		if err != nil {
			// If any glob part is not a valid pattern, don't glob.
			return nil, nil
		}
		rx := regexp.MustCompile("^" + expr + "$")
		var newMatches []string
		for _, dir := range matches {
			newMatches, err = cfg.globDir(base, dir, rx, wantDir, newMatches)
			if err != nil {
				return nil, err
			}
		}
		matches = newMatches
	}
	return matches, nil
}

func (cfg *Config) globDir(base, dir string, rx *regexp.Regexp, wantDir bool, matches []string) ([]string, error) {
	fullDir := dir
	if !filepath.IsAbs(dir) {
		fullDir = filepath.Join(base, dir)
	}
	infos, err := cfg.ReadDir(fullDir)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		name := info.Name()
		if !wantDir {
			// no filtering
		} else if mode := info.Mode(); mode&os.ModeSymlink != 0 {
			// TODO: is there a way to do this without the
			// extra syscall?
			if _, err := cfg.ReadDir(filepath.Join(fullDir, name)); err != nil {
				// symlink pointing to non-directory
				continue
			}
		} else if !mode.IsDir() {
			// definitely not a directory
			continue
		}
		if !strings.HasPrefix(rx.String(), `^\.`) && name[0] == '.' {
			continue
		}
		if rx.MatchString(name) {
			matches = append(matches, pathJoin2(dir, name))
		}
	}
	return matches, nil
}

// ReadFields TODO write doc.
//
// The config specifies shell expansion options; nil behaves the same as an
// empty config.
func ReadFields(cfg *Config, s string, n int, raw bool) []string {
	cfg = prepareConfig(cfg)
	type pos struct {
		start, end int
	}
	var fpos []pos

	runes := make([]rune, 0, len(s))
	infield := false
	esc := false
	for _, r := range s {
		if infield {
			if cfg.ifsRune(r) && (raw || !esc) {
				fpos[len(fpos)-1].end = len(runes)
				infield = false
			}
		} else {
			if !cfg.ifsRune(r) && (raw || !esc) {
				fpos = append(fpos, pos{start: len(runes), end: -1})
				infield = true
			}
		}
		if r == '\\' {
			if raw || esc {
				runes = append(runes, r)
			}
			esc = !esc
			continue
		}
		runes = append(runes, r)
		esc = false
	}
	if len(fpos) == 0 {
		return nil
	}
	if infield {
		fpos[len(fpos)-1].end = len(runes)
	}

	switch {
	case n == 1:
		// include heading/trailing IFSs
		fpos[0].start, fpos[0].end = 0, len(runes)
		fpos = fpos[:1]
	case n != -1 && n < len(fpos):
		// combine to max n fields
		fpos[n-1].end = fpos[len(fpos)-1].end
		fpos = fpos[:n]
	}

	fields := make([]string, len(fpos))
	for i, p := range fpos {
		fields[i] = string(runes[p.start:p.end])
	}
	return fields
}
