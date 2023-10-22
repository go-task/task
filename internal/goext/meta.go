package goext

// NOTE(@andreynering): The lists in this file were copied from:
//
// https://github.com/golang/go/blob/master/src/go/build/syslist.go

func IsKnownOS(str string) bool {
	_, known := knownOS[str]
	return known
}

func IsKnownArch(str string) bool {
	_, known := knownArch[str]
	return known
}

var knownOS = map[string]struct{}{
	"aix":       {},
	"android":   {},
	"darwin":    {},
	"dragonfly": {},
	"freebsd":   {},
	"hurd":      {},
	"illumos":   {},
	"ios":       {},
	"js":        {},
	"linux":     {},
	"nacl":      {},
	"netbsd":    {},
	"openbsd":   {},
	"plan9":     {},
	"solaris":   {},
	"windows":   {},
	"zos":       {},
	"__test__":  {},
}

var knownArch = map[string]struct{}{
	"386":         {},
	"amd64":       {},
	"amd64p32":    {},
	"arm":         {},
	"armbe":       {},
	"arm64":       {},
	"arm64be":     {},
	"loong64":     {},
	"mips":        {},
	"mipsle":      {},
	"mips64":      {},
	"mips64le":    {},
	"mips64p32":   {},
	"mips64p32le": {},
	"ppc":         {},
	"ppc64":       {},
	"ppc64le":     {},
	"riscv":       {},
	"riscv64":     {},
	"s390":        {},
	"s390x":       {},
	"sparc":       {},
	"sparc64":     {},
	"wasm":        {},
}
