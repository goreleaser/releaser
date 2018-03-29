package dist

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDistDoesNotExist(t *testing.T) {
	folder, err := ioutil.TempDir("", "disttest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(
		t,
		Pipe{}.Run(
			&context.Context{
				Config: config.Project{
					Dist: dist,
				},
			},
		),
	)
}

func TestPopulatedDistExists(t *testing.T) {
	folder, err := ioutil.TempDir("", "disttest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	_, err = os.Create(filepath.Join(dist, "mybin"))
	assert.NoError(t, err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: dist,
		},
	}
	assert.Error(t, Pipe{}.Run(ctx))
	ctx.RmDist = true
	assert.NoError(t, Pipe{}.Run(ctx))
	_, err = os.Stat(dist)
	assert.False(t, os.IsExist(err))
}

func TestEmptyDistExists(t *testing.T) {
	folder, err := ioutil.TempDir("", "disttest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	var ctx = &context.Context{
		Config: config.Project{
			Dist: dist,
		},
	}
	assert.NoError(t, Pipe{}.Run(ctx))
	_, err = os.Stat(dist)
	assert.False(t, os.IsNotExist(err))
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestDistIsSneaky(t *testing.T) {
	for _, dist := range []string{"/", "", "/etc/../"} {
		ctx := &context.Context{
			Config: config.Project{
				Dist: dist,
			},
		}
		assert.Error(t, Pipe{}.Run(ctx))
	}
}

func TestDistIsNoSneaky(t *testing.T) {
	for _, dist := range []string{"/home/john", ".", "/tmp/foo/../bar/"} {
		ctx := &context.Context{
			Config: config.Project{
				Dist: dist,
			},
		}
		assert.NoError(t, isSneaky(ctx))
	}
}
