// Package changelog provides the release changelog to goreleaser.
package changelog

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/pipeline"
)

// ErrInvalidSortDirection happens when the sort order is invalid
var ErrInvalidSortDirection = errors.New("invalid sort direction")

// Pipe for checksums
type Pipe struct{}

func (Pipe) String() string {
	return "generating changelog"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.ReleaseNotes != "" {
		return pipeline.Skip("release notes already provided via --release-notes")
	}
	if ctx.Snapshot {
		return pipeline.Skip("not available for snapshots")
	}
	if ctx.Config.Changelog.Extract != "" {
		return extractChangelog(ctx)
	}

	if err := checkSortDirection(ctx.Config.Changelog.Sort); err != nil {
		return err
	}
	entries, err := buildChangelog(ctx)
	if err != nil {
		return err
	}
	ctx.ReleaseNotes = fmt.Sprintf("## Changelog\n\n%v", strings.Join(entries, "\n"))
	return nil
}

func extractChangelog(ctx *context.Context) error {
	fh, err := os.Open(ctx.Config.Changelog.Extract)
	if err != nil {
		return err
	}
	defer fh.Close()
	var relNotes strings.Builder
	scanner := bufio.NewScanner(fh)
	cnt := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "##") {
			cnt++
		}
		if cnt > 1 {
			break
		}
		relNotes.WriteString(line)
		relNotes.WriteString("\n")
	}
	ctx.ReleaseNotes = relNotes.String()
	return nil
}

func checkSortDirection(mode string) error {
	switch mode {
	case "":
		fallthrough
	case "asc":
		fallthrough
	case "desc":
		return nil
	}
	return ErrInvalidSortDirection
}

func buildChangelog(ctx *context.Context) ([]string, error) {
	log, err := getChangelog(ctx.Git.CurrentTag)
	if err != nil {
		return nil, err
	}
	var entries = strings.Split(log, "\n")
	entries = entries[0 : len(entries)-1]
	entries, err = filterEntries(ctx, entries)
	if err != nil {
		return entries, err
	}
	return sortEntries(ctx, entries), nil
}

func filterEntries(ctx *context.Context, entries []string) ([]string, error) {
	for _, filter := range ctx.Config.Changelog.Filters.Exclude {
		r, err := regexp.Compile(filter)
		if err != nil {
			return entries, err
		}
		entries = remove(r, entries)
	}
	return entries, nil
}

func sortEntries(ctx *context.Context, entries []string) []string {
	var direction = ctx.Config.Changelog.Sort
	if direction == "" {
		return entries
	}
	var result = make([]string, len(entries))
	copy(result, entries)
	sort.Slice(result, func(i, j int) bool {
		_, imsg := extractCommitInfo(result[i])
		_, jmsg := extractCommitInfo(result[j])
		if direction == "asc" {
			return strings.Compare(imsg, jmsg) < 0
		}
		return strings.Compare(imsg, jmsg) > 0
	})
	return result
}

func remove(filter *regexp.Regexp, entries []string) (result []string) {
	for _, entry := range entries {
		_, msg := extractCommitInfo(entry)
		if !filter.MatchString(msg) {
			result = append(result, entry)
		}
	}
	return result
}

func extractCommitInfo(line string) (hash, msg string) {
	ss := strings.Split(line, " ")
	return ss[0], strings.Join(ss[1:], " ")
}

func getChangelog(tag string) (string, error) {
	prev, err := previous(tag)
	if err != nil {
		return "", err
	}
	if !prev.Tag {
		return gitLog(prev.SHA, tag)
	}
	return gitLog(fmt.Sprintf("%v..%v", prev.SHA, tag))
}

func gitLog(refs ...string) (string, error) {
	var args = []string{"log", "--pretty=oneline", "--abbrev-commit", "--no-decorate", "--no-color"}
	args = append(args, refs...)
	return git.Run(args...)
}

func previous(tag string) (result ref, err error) {
	result.Tag = true
	result.SHA, err = git.Clean(git.Run("describe", "--tags", "--abbrev=0", tag+"^"))
	if err != nil {
		result.Tag = false
		result.SHA, err = git.Clean(git.Run("rev-list", "--max-parents=0", "HEAD"))
	}
	return
}

type ref struct {
	Tag bool
	SHA string
}
