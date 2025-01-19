package templater

import (
	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/registry/backward"
	"github.com/go-sprout/sprout/registry/checksum"
	"github.com/go-sprout/sprout/registry/conversion"
	"github.com/go-sprout/sprout/registry/encoding"
	"github.com/go-sprout/sprout/registry/env"
	"github.com/go-sprout/sprout/registry/filesystem"
	"github.com/go-sprout/sprout/registry/maps"
	"github.com/go-sprout/sprout/registry/network"
	"github.com/go-sprout/sprout/registry/numeric"
	"github.com/go-sprout/sprout/registry/random"
	"github.com/go-sprout/sprout/registry/reflect"
	"github.com/go-sprout/sprout/registry/regexp"
	"github.com/go-sprout/sprout/registry/slices"
	"github.com/go-sprout/sprout/registry/std"
	"github.com/go-sprout/sprout/registry/strings"
	"github.com/go-sprout/sprout/registry/time"
	taskfunc "github.com/go-task/task/v3/internal/templater/taskfunc"
	"github.com/go-task/template"
)

var templateFuncs template.FuncMap

// BACKWARDS COMPATIBILITY
// The following functions are provided for backwards compatibility with the
// original sprig methods. They are not recommended for use in new code.
var bc_registerSprigFuncs = sprout.FunctionAliasMap{
	"dateModify":     []string{"date_modify"},                   // ! Deprecated: Should use dateModify instead
	"dateInZone":     []string{"date_in_zone"},                  // ! Deprecated: Should use dateInZone instead
	"mustDateModify": []string{"must_date_modify"},              // ! Deprecated: Should use mustDateModify instead
	"ellipsis":       []string{"abbrev"},                        // ! Deprecated: Should use ellipsis instead
	"ellipsisBoth":   []string{"abbrevboth"},                    // ! Deprecated: Should use ellipsisBoth instead
	"trimAll":        []string{"trimall"},                       // ! Deprecated: Should use trimAll instead
	"append":         []string{"push"},                          // ! Deprecated: Should use append instead
	"mustAppend":     []string{"mustPush"},                      // ! Deprecated: Should use mustAppend instead
	"list":           []string{"tuple"},                         // ! Deprecated: Should use list instead
	"max":            []string{"biggest"},                       // ! Deprecated: Should use max instead
	"toUpper":        []string{"upper", "toupper", "uppercase"}, // ! Deprecated: Should use toUpper instead
	"toLower":        []string{"lower", "tolower", "lowercase"}, // ! Deprecated: Should use toLower instead
	"add":            []string{"addf"},                          // ! Deprecated: Should use add instead
	"add1":           []string{"add1f"},                         // ! Deprecated: Should use add1 instead
	"sub":            []string{"subf"},                          // ! Deprecated: Should use sub instead
	"toTitleCase":    []string{"title", "titlecase"},            // ! Deprecated: Should use toTitleCase instead
	"toPascalCase":   []string{"camelcase"},                     // ! Deprecated: Should use toPascalCase instead
	"toSnakeCase":    []string{"snake", "snakecase"},            // ! Deprecated: Should use toSnakeCase instead
	"toKebabCase":    []string{"kebab", "kebabcase"},            // ! Deprecated: Should use toKebabCase instead
	"swapCase":       []string{"swapcase"},                      // ! Deprecated: Should use swapCase instead
	"base64Encode":   []string{"b64enc"},                        // ! Deprecated: Should use base64Encode instead
	"base64Decode":   []string{"b64dec"},                        // ! Deprecated: Should use base64Decode instead
	"base32Encode":   []string{"b32enc"},                        // ! Deprecated: Should use base32Encode instead
	"base32Decode":   []string{"b32dec"},                        // ! Deprecated: Should use base32Decode instead
	"pathBase":       []string{"base"},                          // ! Deprecated: Should use pathBase instead
	"pathDir":        []string{"dir"},                           // ! Deprecated: Should use pathDir instead
	"pathExt":        []string{"ext"},                           // ! Deprecated: Should use pathExt instead
	"pathClean":      []string{"clean"},                         // ! Deprecated: Should use pathClean instead
	"pathIsAbs":      []string{"isAbs"},                         // ! Deprecated: Should use pathIsAbs instead
	"expandEnv":      []string{"expandenv"},                     // ! Deprecated: Should use expandEnv instead
	"dateAgo":        []string{"ago"},                           // ! Deprecated: Should use dateAgo instead
	"strSlice":       []string{"toStrings"},                     // ! Deprecated: Should use strSlice instead
	"toInt":          []string{"int", "atoi"},                   // ! Deprecated: Should use toInt instead
	"toInt64":        []string{"int64"},                         // ! Deprecated: Should use toInt64 instead
	"toFloat64":      []string{"float64"},                       // ! Deprecated: Should use toFloat64 instead
	"toOctal":        []string{"toDecimal"},                     // ! Deprecated: Should use toOctal instead
}

// prepareBackwardCompatibilityOpts returns a slice of sprout.HandlerOption that
// registers old sprig function names as aliases for their new names and adds
// deprecation notices to the old names. This is required to ensure backward
// compatibility with older versions of go-task that used the old sprig names.
func prepareBackwardCompatibilityOpts() []sprout.HandlerOption[*sprout.DefaultHandler] {
	var opts = make([]sprout.HandlerOption[*sprout.DefaultHandler], (len(bc_registerSprigFuncs)+8)*2) // 8 represents multiple aliases for specific functions

	for originalFunction, aliases := range bc_registerSprigFuncs {
		opts = append(opts, sprout.WithAlias(originalFunction, aliases...))

		for _, alias := range aliases {
			opts = append(opts, sprout.WithNotices(sprout.NewDeprecatedNotice(alias, "please use `"+originalFunction+"` instead")))
		}
	}

	return opts
}

// \ BACKWARDS COMPATIBILITY

func init() {
	opts := []sprout.HandlerOption[*sprout.DefaultHandler]{
		sprout.WithRegistries(
			// Library registries
			backward.NewRegistry(),
			checksum.NewRegistry(),
			conversion.NewRegistry(),
			encoding.NewRegistry(),
			env.NewRegistry(),
			filesystem.NewRegistry(),
			maps.NewRegistry(),
			network.NewRegistry(),
			numeric.NewRegistry(),
			random.NewRegistry(),
			reflect.NewRegistry(),
			regexp.NewRegistry(),
			slices.NewRegistry(),
			std.NewRegistry(),
			strings.NewRegistry(),
			time.NewRegistry(),
			// Own registry
			taskfunc.NewRegistry(),
		),
	}
	opts = append(opts, prepareBackwardCompatibilityOpts()...)
	handler := sprout.New(opts...)

	templateFuncs = template.FuncMap(handler.Build())
}
