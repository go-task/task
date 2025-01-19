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

func init() {
	handler := sprout.New(
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
	)

	templateFuncs = template.FuncMap(handler.Build())
}
