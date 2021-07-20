package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestArchiveFiles_justString(t *testing.T) {
	var actual Archive

	err := yaml.UnmarshalStrict([]byte(`
files:
- ./script.sh
`), &actual)
	require.NoError(t, err)
	require.Equal(t, []File{
		{
			Source:      "./script.sh",
			Destination: "./script.sh",
		},
	}, actual.Files)
}

func TestArchiveFiles_stringFiles(t *testing.T) {
	var actual Archive

	err := yaml.UnmarshalStrict([]byte(`
files:
- ./script.sh
- src: ./foo
  dst: ./bar
  info:
    owner: carlos
    group: users
`), &actual)
	require.NoError(t, err)
	require.Equal(t, []File{
		{
			Source:      "./script.sh",
			Destination: "./script.sh",
		},
		{
			Source:      "./foo",
			Destination: "./bar",
			Info: FileInfo{
				Owner: "carlos",
				Group: "users",
			},
		},
	}, actual.Files)
}

func TestArchiveFiles_complex(t *testing.T) {
	var actual Archive
	now := time.Now().UTC().Truncate(time.Second)

	// 2021-07-17T15:14:10.264931-03:00

	err := yaml.UnmarshalStrict([]byte(fmt.Sprintf(`
files:
- src: ./foo
  dst: ./bar
  info:
    owner: carlos
    group: users
- src: ./foobar
  dst: ./barzaz
  info:
    owner: carlos
    group: users
    mode: 0644
    mtime: "%s"
`, now.Format(time.RFC3339Nano))), &actual)
	require.NoError(t, err)

	require.Equal(t, []File{
		{
			Source:      "./foo",
			Destination: "./bar",
			Info: FileInfo{
				Owner: "carlos",
				Group: "users",
			},
		},
		{
			Source:      "./foobar",
			Destination: "./barzaz",
			Info: FileInfo{
				Owner: "carlos",
				Group: "users",
				Mode:  0644,
				MTime: now,
			},
		},
	}, actual.Files)
}
