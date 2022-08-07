package sandbox

import (
	"os"

	"github.com/landlock-lsm/go-landlock/landlock"
	llsys "github.com/landlock-lsm/go-landlock/landlock/syscall"
)

type pathException struct {
	Paths           []string
	PermittedAccess []string
}

func WithSandbox(sources []string, generates []string) error {
	sandbox, err := landlock.NewConfig(defaultAccessFSSet())
	if err != nil {
		return err
	}

	defaultAllowRW := []string{
		".task",
		"/tmp",
	}

	var allowList []landlock.PathOpt
	allowList = append(allowList, landlock.RODirs(defaultAllowRW...))
	allowList = append(allowList, parseSources(sources)...)
	allowList = append(allowList, parseGenerates(generates)...)

	return sandbox.RestrictPaths(allowList...)
}

func defaultAccessFSSet() landlock.AccessFSSet {
	var restricted landlock.AccessFSSet

	table := map[string]landlock.AccessFSSet{
		"execute":     llsys.AccessFSExecute,
		"write_file":  llsys.AccessFSWriteFile,
		"read_file":   llsys.AccessFSReadFile,
		"read_dir":    llsys.AccessFSReadDir,
		"remove_dir":  llsys.AccessFSRemoveDir,
		"remove_file": llsys.AccessFSRemoveFile,
		"make_char":   llsys.AccessFSMakeChar,
		"make_dir":    llsys.AccessFSMakeDir,
		"make_reg":    llsys.AccessFSMakeReg,
		"make_sock":   llsys.AccessFSMakeSock,
		"make_fifo":   llsys.AccessFSMakeFifo,
		"make_block":  llsys.AccessFSMakeBlock,
		"make_sym":    llsys.AccessFSMakeSym,
	}

	// TODO: Might wanna make those configurable
	for _, accessFSSet := range table {
		restricted |= accessFSSet
	}

	return restricted
}

func parseSources(sources []string) []landlock.PathOpt {
	var opts []landlock.PathOpt

	for _, source := range sources {
		if stat, err := os.Stat(source); err == nil {
			if stat.IsDir() {
				opts = append(opts, landlock.RODirs(source))
			}

			opts = append(opts, landlock.ROFiles(source))
		}
	}

	return opts
}

func parseGenerates(generates []string) []landlock.PathOpt {
	var opts []landlock.PathOpt

	for _, generate := range generates {
		if stat, err := os.Stat(generate); err == nil {
			if stat.IsDir() {
				opts = append(opts, landlock.RWDirs(generate))
			}

			opts = append(opts, landlock.RWFiles(generate))
		}
	}

	return opts
}
