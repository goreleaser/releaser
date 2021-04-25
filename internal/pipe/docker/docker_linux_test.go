package docker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinkTwoLevelDirectory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	srcLevel2 := filepath.Join(srcDir, "level2")
	const testFile = "test"

	require.NoError(t, os.Mkdir(srcLevel2, 0o755))
	require.NoError(t, ioutil.WriteFile(filepath.Join(srcDir, testFile), []byte("foo"), 0o644))
	require.NoError(t, ioutil.WriteFile(filepath.Join(srcLevel2, testFile), []byte("foo"), 0o644))

	require.NoError(t, link(srcDir, dstDir))

	require.Equal(t, inode(filepath.Join(srcDir, testFile)), inode(filepath.Join(dstDir, testFile)))
	require.Equal(t, inode(filepath.Join(srcLevel2, testFile)), inode(filepath.Join(dstDir, "level2", testFile)))
}

func inode(file string) uint64 {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return 0
	}
	stat := fileInfo.Sys().(*syscall.Stat_t)
	return stat.Ino
}

