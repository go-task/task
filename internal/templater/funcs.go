package templater

import (
	"log/slog"
	"maps"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/registry/backward"
	"github.com/go-sprout/sprout/registry/checksum"
	"github.com/go-sprout/sprout/registry/conversion"
	"github.com/go-sprout/sprout/registry/encoding"
	"github.com/go-sprout/sprout/registry/env"
	"github.com/go-sprout/sprout/registry/filesystem"
	sproutmaps "github.com/go-sprout/sprout/registry/maps"
	"github.com/go-sprout/sprout/registry/numeric"
	"github.com/go-sprout/sprout/registry/random"
	"github.com/go-sprout/sprout/registry/reflect"
	"github.com/go-sprout/sprout/registry/regexp"
	"github.com/go-sprout/sprout/registry/slices"
	"github.com/go-sprout/sprout/registry/std"
	sproutstrings "github.com/go-sprout/sprout/registry/strings"
	sprouttime "github.com/go-sprout/sprout/registry/time"
	"github.com/go-sprout/sprout/registry/uniqueid"

	"github.com/go-task/template"

	"github.com/go-task/task/v3/internal/templater/taskfuncs"
)

var templateFuncs template.FuncMap

// legacySprigAliases maps the function names Task exposed through slim-sprig
// onto their sprout equivalents. Only names slim-sprig actually shipped are
// listed — sprout carries a wider legacy set, but Task never exposed those and
// should not start now.
var legacySprigAliases = sprout.FunctionAliasMap{
	"dateModify":   {"date_modify", "must_date_modify"},
	"dateInZone":   {"date_in_zone"},
	"dateAgo":      {"ago"},
	"trimAll":      {"trimall"},
	"append":       {"push", "mustPush"},
	"list":         {"tuple"},
	"max":          {"biggest"},
	"toUpper":      {"upper"},
	"toLower":      {"lower"},
	"toTitleCase":  {"title"},
	"base64Encode": {"b64enc"},
	"base64Decode": {"b64dec"},
	"base32Encode": {"b32enc"},
	"base32Decode": {"b32dec"},
	"pathBase":     {"base"},
	"pathDir":      {"dir"},
	"pathExt":      {"ext"},
	"pathClean":    {"clean"},
	"pathIsAbs":    {"isAbs"},
	"expandEnv":    {"expandenv"},
	"strSlice":     {"toStrings"},
	"toInt":        {"int", "atoi"},
	"toInt64":      {"int64"},
	"toFloat64":    {"float64"},
	"toOctal":      {"toDecimal"},
}

func init() {
	handler := sprout.New(
		sprout.WithLogger(slog.New(slog.DiscardHandler)),
		sprout.WithRegistries(
			taskfuncs.NewRegistry(),
			backward.NewRegistry(),
			checksum.NewRegistry(),
			conversion.NewRegistry(),
			encoding.NewRegistry(),
			env.NewRegistry(),
			filesystem.NewRegistry(),
			sproutmaps.NewRegistry(),
			numeric.NewRegistry(),
			random.NewRegistry(),
			reflect.NewRegistry(),
			regexp.NewRegistry(),
			slices.NewRegistry(),
			std.NewRegistry(),
			sproutstrings.NewRegistry(),
			sprouttime.NewRegistry(),
			uniqueid.NewRegistry(),
		),
	)

	for original, aliases := range legacySprigAliases {
		_ = sprout.WithAlias(original, aliases...)(handler)
		for _, alias := range aliases {
			_ = sprout.WithNotices(sprout.NewDeprecatedNotice(alias, "please use `"+original+"` instead"))(handler)
		}
	}

	templateFuncs = template.FuncMap(handler.Build())
	maps.Copy(templateFuncs, taskfuncs.Overrides())
	maps.Copy(templateFuncs, sprigSignatureShims(handler))
}
