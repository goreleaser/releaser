package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNotAGitFolder(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), "fatal: Not a git repository (or any of the parent directories): .git\n")
}

func TestSingleCommit(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
}

func TestNewRepository(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	// TODO: improve this error handling
	assert.Contains(t, Pipe{}.Run(ctx).Error(), `fatal: ambiguous argument 'HEAD'`)
}

func TestNoTagsSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "SNAPSHOT-{{.Commit}}",
		},
	})
	ctx.Snapshot = true
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Contains(t, ctx.Version, "SNAPSHOT-")
}

func TestNoTagsSnapshotInvalidTemplate(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{",
		},
	})
	ctx.Snapshot = true
	assert.EqualError(t, Pipe{}.Run(ctx), `failed to generate snapshot name: template: snapshot:1: unexpected unclosed action in command`)
}

// TestNoTagsNoSnapshot covers the situation where a repository
// only contains simple commits and no tags. In this case you have
// to set the --snapshot flag otherwise an error is returned.
func TestNoTagsNoSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	ctx.Snapshot = false
	assert.EqualError(t, Pipe{}.Run(ctx), `git doesn't contain any tags. Either add a tag or use --snapshot`)
}

func TestInvalidTagFormat(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "sadasd")
	var ctx = context.New(config.Project{})
	assert.EqualError(t, Pipe{}.Run(ctx), "sadasd is not in a valid version format")
	assert.Equal(t, "sadasd", ctx.Git.CurrentTag)
}

func TestDirty(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	assert.NoError(t, err)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	assert.NoError(t, ioutil.WriteFile(dummy.Name(), []byte("lorem ipsum"), 0644))
	err = Pipe{}.Run(context.New(config.Project{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git is currently in a dirty state:")
}

func TestTagIsNotLastCommit(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	err := Pipe{}.Run(context.New(config.Project{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git tag v0.0.1 was not made against commit")
}

func TestValidState(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{})
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "v0.0.2", ctx.Git.CurrentTag)
}

func TestSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	assert.NoError(t, Pipe{}.Run(ctx))
}

// TODO: missing a test case for a dirty git tree and snapshot
