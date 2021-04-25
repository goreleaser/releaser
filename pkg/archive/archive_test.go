package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	var folder = t.TempDir()
	empty, err := os.Create(filepath.Join(folder, "empty.txt"))
	require.NoError(t, err)
	require.NoError(t, empty.Close())
	require.NoError(t, os.Mkdir(folder+"/folder-inside", 0755))

	for _, format := range []string{"tar.gz", "zip", "gz", "tar.xz", "willbeatargzanyway"} {
		format := format
		t.Run(format, func(t *testing.T) {
			file, err := os.Create(filepath.Join(folder, "folder."+format))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, file.Close())
			})
			var archive = New(file)
			t.Cleanup(func() {
				require.NoError(t, archive.Close())
				require.NoError(t, file.Close())
			})
			require.NoError(t, archive.Add("empty.txt", empty.Name()))
			require.Error(t, archive.Add("dont.txt", empty.Name()+"_nope"))
		})
	}
}
