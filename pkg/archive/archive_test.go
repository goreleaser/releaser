package archive

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	folder := t.TempDir()
	empty, err := os.Create(folder + "/empty.txt")
	require.NoError(t, err)
	require.NoError(t, empty.Close())
	require.NoError(t, os.Mkdir(folder+"/folder-inside", 0o755))

	for _, format := range []string{"tar.gz", "zip", "gz", "tar.xz", "willbeatargzanyway"} {
		format := format
		t.Run(format, func(t *testing.T) {
			file, err := os.Create(folder + "/folder." + format)
			require.NoError(t, err)
			archive := New(file)
			t.Cleanup(func() {
				require.NoError(t, archive.Close())
				require.NoError(t, file.Close())
			})
			require.NoError(t, archive.Add(config.File{
				Source:      empty.Name(),
				Destination: "empty.txt",
			}))
			require.Error(t, archive.Add(config.File{
				Source:      empty.Name() + "_nope",
				Destination: "dont.txt",
			}))
		})
	}
}
