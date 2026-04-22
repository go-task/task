package output

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/go-task/task/v3/internal/templater"
)

// GitLab renders a task's output wrapped in [GitLab CI collapsible
// section markers]. Section IDs are generated automatically so that
// start and end markers always match and stay unique per invocation.
//
// [GitLab CI collapsible section markers]: https://docs.gitlab.com/ci/jobs/job_logs/#create-custom-collapsible-sections
type GitLab struct {
	Collapsed bool
	ErrorOnly bool
}

func (g GitLab) WrapWriter(stdOut, _ io.Writer, _ string, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	header := ""
	if cache != nil {
		header = templater.Replace("{{.TASK}}", cache)
	}
	if header == "" {
		header = "task"
	}

	id := fmt.Sprintf("%s_%s", gitlabSectionSlug(header), uuid.New().String()[:8])

	gw := &gitlabWriter{
		writer:    stdOut,
		id:        id,
		header:    header,
		collapsed: g.Collapsed,
		startTS:   time.Now().Unix(),
	}

	return gw, gw, func(err error) error {
		if g.ErrorOnly && err == nil {
			return nil
		}
		return gw.close()
	}
}

type gitlabWriter struct {
	writer    io.Writer
	buff      bytes.Buffer
	id        string
	header    string
	collapsed bool
	startTS   int64
}

func (gw *gitlabWriter) Write(p []byte) (int, error) {
	return gw.buff.Write(p)
}

func (gw *gitlabWriter) close() error {
	if gw.buff.Len() == 0 {
		return nil
	}

	var b bytes.Buffer
	b.WriteString(gitlabSectionStart(gw.startTS, gw.id, gw.header, gw.collapsed))
	if _, err := io.Copy(&b, &gw.buff); err != nil {
		return err
	}
	b.WriteString(gitlabSectionEnd(time.Now().Unix(), gw.id))

	_, err := io.Copy(gw.writer, &b)
	return err
}

func gitlabSectionStart(ts int64, id, header string, collapsed bool) string {
	options := ""
	if collapsed {
		options = "[collapsed=true]"
	}
	return fmt.Sprintf("\x1b[0Ksection_start:%d:%s%s\r\x1b[0K%s\n", ts, id, options, header)
}

func gitlabSectionEnd(ts int64, id string) string {
	return fmt.Sprintf("\x1b[0Ksection_end:%d:%s\r\x1b[0K\n", ts, id)
}

var gitlabSlugDisallowed = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

func gitlabSectionSlug(s string) string {
	return gitlabSlugDisallowed.ReplaceAllString(s, "_")
}
