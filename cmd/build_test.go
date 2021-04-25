package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	setup(t)
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestBuildSingleTarget(t *testing.T) {
	setup(t)
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated", "--single-target"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestBuildInvalidConfig(t *testing.T) {
	setup(t)
	createFile(t, "goreleaser.yml", "foo: bar")
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2", "--deprecated"})
	require.EqualError(t, cmd.cmd.Execute(), "yaml: unmarshal errors:\n  line 1: field foo not found in type config.Project")
}

func TestBuildBrokenProject(t *testing.T) {
	folder := setup(t)
	t.Cleanup(func() {
		filepath.Walk(folder, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			t.Log("removing", path)
			return os.Remove(path)
		})
	})
	createFile(t, "main.go", "not a valid go file")
	cmd := newBuildCmd()
	cmd.cmd.SetArgs([]string{"--snapshot", "--timeout=1m", "--parallelism=2"})
	require.EqualError(t, cmd.cmd.Execute(), "failed to parse dir: .: main.go:1:1: expected 'package', found not")
}

func TestBuildFlags(t *testing.T) {
	setup := func(opts buildOpts) *context.Context {
		ctx := context.New(config.Project{})
		require.NoError(t, setupBuildContext(ctx, opts))
		return ctx
	}

	t.Run("snapshot", func(t *testing.T) {
		ctx := setup(buildOpts{
			snapshot: true,
		})
		require.True(t, ctx.Snapshot)
		require.True(t, ctx.SkipValidate)
		require.True(t, ctx.SkipTokenCheck)
	})

	t.Run("skips", func(t *testing.T) {
		ctx := setup(buildOpts{
			skipValidate:  true,
			skipPostHooks: true,
		})
		require.True(t, ctx.SkipValidate)
		require.True(t, ctx.SkipPostBuildHooks)
		require.True(t, ctx.SkipTokenCheck)
	})

	t.Run("parallelism", func(t *testing.T) {
		require.Equal(t, 1, setup(buildOpts{
			parallelism: 1,
		}).Parallelism)
	})

	t.Run("rm dist", func(t *testing.T) {
		require.True(t, setup(buildOpts{
			rmDist: true,
		}).RmDist)
	})

	t.Run("single-target", func(t *testing.T) {
		opts := buildOpts{
			singleTarget: true,
		}

		t.Run("runtime", func(t *testing.T) {
			result := setup(opts)
			require.Equal(t, []string{runtime.GOOS}, result.Config.Builds[0].Goos)
			require.Equal(t, []string{runtime.GOARCH}, result.Config.Builds[0].Goarch)
		})

		t.Run("from env", func(t *testing.T) {
			os.Setenv("GOOS", "linux")
			os.Setenv("GOARCH", "arm64")
			t.Cleanup(func() {
				os.Unsetenv("GOOS")
				os.Unsetenv("GOARCH")
			})
			result := setup(opts)
			require.Equal(t, []string{"linux"}, result.Config.Builds[0].Goos)
			require.Equal(t, []string{"arm64"}, result.Config.Builds[0].Goarch)
		})
	})

	t.Run("id", func(t *testing.T) {
		t.Run("match", func(t *testing.T) {
			ctx := context.New(config.Project{
				Builds: []config.Build{
					{
						ID: "default",
					},
					{
						ID: "foo",
					},
				},
			})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				id: "foo",
			}))
		})

		t.Run("dont match", func(t *testing.T) {
			ctx := context.New(config.Project{
				Builds: []config.Build{
					{
						ID: "foo",
					},
					{
						ID: "bazz",
					},
				},
			})
			require.EqualError(t, setupBuildContext(ctx, buildOpts{
				id: "bar",
			}), "no builds with id 'bar'")
		})

		t.Run("default config", func(t *testing.T) {
			ctx := context.New(config.Project{})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				id: "aaa",
			}))
		})

		t.Run("single build config", func(t *testing.T) {
			ctx := context.New(config.Project{
				Builds: []config.Build{
					{
						ID: "foo",
					},
				},
			})
			require.NoError(t, setupBuildContext(ctx, buildOpts{
				id: "not foo but doesnt matter",
			}))
		})
	})
}
