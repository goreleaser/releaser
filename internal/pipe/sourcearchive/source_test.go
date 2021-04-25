package sourcearchive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	for _, format := range []string{"tar.gz", "tar", "zip"} {
		t.Run(format, func(t *testing.T) {
			var tmp = testlib.Mktmp(t)
			require.NoError(t, os.Mkdir("dist", 0744))

			testlib.GitInit(t)
			require.NoError(t, os.WriteFile("code.txt", []byte("not really code"), 0655))
			require.NoError(t, os.WriteFile("README.md", []byte("# my dope fake project"), 0655))
			testlib.GitAdd(t)
			testlib.GitCommit(t, "feat: first")

			var ctx = context.New(config.Project{
				ProjectName: "foo",
				Dist:        "dist",
				Source: config.Source{
					Format:  format,
					Enabled: true,
				},
			})
			ctx.Git.FullCommit = "HEAD"
			ctx.Version = "1.0.0"

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Run(ctx))

			var artifacts = ctx.Artifacts.List()
			require.Len(t, artifacts, 1)
			require.Equal(t, artifact.Artifact{
				Type: artifact.UploadableSourceArchive,
				Name: "foo-1.0.0." + format,
				Path: filepath.Join("dist", "foo-1.0.0." + format),
				Extra: map[string]interface{}{
					"Format": format,
				},
			}, *artifacts[0])
			stat, err := os.Stat(filepath.Join(tmp, "dist", "foo-1.0.0."+format))
			require.NoError(t, err)
			require.Greater(t, stat.Size(), int64(100))
		})
	}
}

func TestDefault(t *testing.T) {
	var ctx = context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, config.Source{
		NameTemplate: "{{ .ProjectName }}-{{ .Version }}",
		Format:       "tar.gz",
	}, ctx.Config.Source)
}

func TestInvalidNameTemplate(t *testing.T) {
	var ctx = context.New(config.Project{
		Source: config.Source{
			Enabled:      true,
			NameTemplate: "{{ .foo }-asdda",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), "template: tmpl:1: unexpected \"}\" in operand")
}

func TestDisabled(t *testing.T) {
	testlib.AssertSkipped(t, Pipe{}.Run(context.New(config.Project{})))
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
