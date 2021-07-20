package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/archive"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func createFakeBinary(t *testing.T, dist, arch, bin string) {
	t.Helper()
	path := filepath.Join(dist, arch, bin)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestRunPipe(t *testing.T) {
	folder := testlib.Mktmp(t)
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(t *testing.T) {
			dist := filepath.Join(folder, format+"_dist")
			require.NoError(t, os.Mkdir(dist, 0o755))
			for _, arch := range []string{"darwinamd64", "linux386", "linuxarm7", "linuxmipssoftfloat"} {
				createFakeBinary(t, dist, arch, "bin/mybin")
			}
			createFakeBinary(t, dist, "windowsamd64", "bin/mybin.exe")
			for _, tt := range []string{"darwin", "linux", "windows"} {
				f, err := os.Create(filepath.Join(folder, fmt.Sprintf("README.%s.md", tt)))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			require.NoError(t, os.MkdirAll(filepath.Join(folder, "foo", "bar", "foobar"), 0o755))
			f, err := os.Create(filepath.Join(filepath.Join(folder, "foo", "bar", "foobar", "blah.txt")))
			require.NoError(t, err)
			require.NoError(t, f.Close())
			ctx := context.New(
				config.Project{
					Dist:        dist,
					ProjectName: "foobar",
					Archives: []config.Archive{
						{
							ID:           "myid",
							Builds:       []string{"default"},
							NameTemplate: defaultNameTemplate,
							Files: []config.File{
								{Source: "README.{{.Os}}.*"},
								{Source: "./foo/**/*"},
							},
							FormatOverrides: []config.FormatOverride{
								{
									Goos:   "windows",
									Format: "zip",
								},
							},
						},
					},
				},
			)
			darwinBuild := &artifact.Artifact{
				Goos:   "darwin",
				Goarch: "amd64",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "darwinamd64", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "bin/mybin",
					"ID":     "default",
				},
			}
			linux386Build := &artifact.Artifact{
				Goos:   "linux",
				Goarch: "386",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "bin/mybin",
					"ID":     "default",
				},
			}
			linuxArmBuild := &artifact.Artifact{
				Goos:   "linux",
				Goarch: "arm",
				Goarm:  "7",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linuxarm7", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "bin/mybin",
					"ID":     "default",
				},
			}
			linuxMipsBuild := &artifact.Artifact{
				Goos:   "linux",
				Goarch: "mips",
				Gomips: "softfloat",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linuxmipssoftfloat", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "mybin",
					"ID":     "default",
				},
			}
			windowsBuild := &artifact.Artifact{
				Goos:   "windows",
				Goarch: "amd64",
				Name:   "bin/mybin.exe",
				Path:   filepath.Join(dist, "windowsamd64", "bin", "mybin.exe"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary":    "mybin",
					"Extension": ".exe",
					"ID":        "default",
				},
			}
			ctx.Artifacts.Add(darwinBuild)
			ctx.Artifacts.Add(linux386Build)
			ctx.Artifacts.Add(linuxArmBuild)
			ctx.Artifacts.Add(linuxMipsBuild)
			ctx.Artifacts.Add(windowsBuild)
			ctx.Version = "0.0.1"
			ctx.Git.CurrentTag = "v0.0.1"
			ctx.Config.Archives[0].Format = format
			require.NoError(t, Pipe{}.Run(ctx))
			archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()
			for _, arch := range archives {
				require.Equal(t, "myid", arch.Extra["ID"].(string), "all archives should have the archive ID set")
			}
			require.Len(t, archives, 5)
			// TODO: should verify the artifact fields here too

			if format == "tar.gz" {
				// Check archive contents
				for name, os := range map[string]string{
					"foobar_0.0.1_darwin_amd64.tar.gz":         "darwin",
					"foobar_0.0.1_linux_386.tar.gz":            "linux",
					"foobar_0.0.1_linux_armv7.tar.gz":          "linux",
					"foobar_0.0.1_linux_mips_softfloat.tar.gz": "linux",
				} {
					require.Equal(
						t,
						[]string{
							fmt.Sprintf("README.%s.md", os),
							"foo/bar/foobar/blah.txt",
							"bin/mybin",
						},
						tarFiles(t, filepath.Join(dist, name)),
					)
				}
			}
			if format == "zip" {
				require.Equal(
					t,
					[]string{
						"README.windows.md",
						"foo/bar/foobar/blah.txt",
						"bin/mybin.exe",
					},
					zipFiles(t, filepath.Join(dist, "foobar_0.0.1_windows_amd64.zip")),
				)
			}
		})
	}
}

func TestRunPipeDifferentBinaryCount(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	for _, arch := range []string{"darwinamd64", "linuxamd64"} {
		createFakeBinary(t, dist, arch, "bin/mybin")
	}
	createFakeBinary(t, dist, "darwinamd64", "bin/foobar")
	ctx := context.New(config.Project{
		Dist:        dist,
		ProjectName: "foobar",
		Archives: []config.Archive{
			{
				ID:           "myid",
				Format:       "tar.gz",
				Builds:       []string{"default", "foobar"},
				NameTemplate: defaultNameTemplate,
			},
		},
	})
	darwinBuild := &artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "darwinamd64", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "bin/mybin",
			"ID":     "default",
		},
	}
	darwinBuild2 := &artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "bin/foobar",
		Path:   filepath.Join(dist, "darwinamd64", "bin", "foobar"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "bin/foobar",
			"ID":     "foobar",
		},
	}
	linuxArmBuild := &artifact.Artifact{
		Goos:   "linux",
		Goarch: "amd64",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linuxamd64", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "bin/mybin",
			"ID":     "default",
		},
	}

	ctx.Artifacts.Add(darwinBuild)
	ctx.Artifacts.Add(darwinBuild2)
	ctx.Artifacts.Add(linuxArmBuild)
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"

	t.Run("check enabled", func(t *testing.T) {
		ctx.Config.Archives[0].AllowDifferentBinaryCount = false
		require.EqualError(t, Pipe{}.Run(ctx), "invalid archive: 0: "+ErrArchiveDifferentBinaryCount.Error())
	})

	t.Run("check disabled", func(t *testing.T) {
		ctx.Config.Archives[0].AllowDifferentBinaryCount = true
		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestRunPipeNoBinaries(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		Dist:        dist,
		ProjectName: "foobar",
		Archives:    []config.Archive{{}},
	})
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	require.NoError(t, Pipe{}.Run(ctx))
}

func zipFiles(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	info, err := f.Stat()
	require.NoError(t, err)
	r, err := zip.NewReader(f, info.Size())
	require.NoError(t, err)
	paths := make([]string, len(r.File))
	for i, zf := range r.File {
		paths[i] = zf.Name
	}
	return paths
}

func tarFiles(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()
	r := tar.NewReader(gr)
	var paths []string
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		paths = append(paths, next.Name)
	}
	return paths
}

func TestRunPipeBinary(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Format:       "binary",
					NameTemplate: defaultBinaryNameTemplate,
					Builds:       []string{"default"},
				},
			},
		},
	)
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"Ext":    ".exe",
			"ID":     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	binaries := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
	darwin := binaries.Filter(artifact.ByGoos("darwin")).List()[0]
	windows := binaries.Filter(artifact.ByGoos("windows")).List()[0]
	require.Equal(t, "mybin_0.0.1_darwin_amd64", darwin.Name)
	require.Equal(t, "mybin_0.0.1_windows_amd64.exe", windows.Name)
	require.Len(t, binaries.List(), 2)
}

func TestRunPipeDistRemoved(t *testing.T) {
	ctx := context.New(
		config.Project{
			Dist: "/tmp/path/to/nope",
			Archives: []config.Archive{
				{
					NameTemplate: "nope",
					Format:       "zip",
					Builds:       []string{"default"},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join("/tmp/path/to/nope", "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary":    "mybin",
			"Extension": ".exe",
			"ID":        "default",
		},
	})
	// not checking on error msg because it may change depending on OS/version
	require.Error(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidGlob(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []config.File{
						{Source: "[x-]"},
					},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to find files to archive: globbing failed for pattern [x-]: compile glob pattern: unexpected end of input`)
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo{{ .fff }",
					Format:       "zip",
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeInvalidFilesNameTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []config.File{
						{Source: "{{.asdsd}"},
					},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to find files to archive: failed to apply template {{.asdsd}: template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeInvalidWrapInDirectoryTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:          []string{"default"},
					NameTemplate:    "foo",
					WrapInDirectory: "foo{{ .fff }",
					Format:          "zip",
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWrap(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:          []string{"default"},
					NameTemplate:    "foo",
					WrapInDirectory: "foo_{{ .Os }}",
					Format:          "tar.gz",
					Replacements: map[string]string{
						"darwin": "macOS",
					},
					Files: []config.File{
						{Source: "README.*"},
					},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))

	archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()
	require.Len(t, archives, 1)
	require.Equal(t, "foo_macOS", archives[0].ExtraOr("WrappedIn", ""))

	// Check archive contents
	f, err = os.Open(filepath.Join(dist, "foo.tar.gz"))
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer func() { require.NoError(t, gr.Close()) }()
	r := tar.NewReader(gr)
	for _, n := range []string{"README.md", "mybin"} {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Equal(t, filepath.Join("foo_macOS", n), h.Name)
	}
}

func TestDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Archives: []config.Archive{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.NotEmpty(t, ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "tar.gz", ctx.Config.Archives[0].Format)
	require.NotEmpty(t, ctx.Config.Archives[0].Files)
}

func TestDefaultSet(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []config.File{
						{Source: "foo"},
					},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "zip", ctx.Config.Archives[0].Format)
	require.Equal(t, config.File{Source: "foo"}, ctx.Config.Archives[0].Files[0])
}

func TestDefaultFormatBinary(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					Format: "binary",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultBinaryNameTemplate, ctx.Config.Archives[0].NameTemplate)
}

func TestFormatFor(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					Builds: []string{"default"},
					Format: "tar.gz",
					FormatOverrides: []config.FormatOverride{
						{
							Goos:   "windows",
							Format: "zip",
						},
					},
				},
			},
		},
	}
	require.Equal(t, "zip", packageFormat(ctx.Config.Archives[0], "windows"))
	require.Equal(t, "tar.gz", packageFormat(ctx.Config.Archives[0], "linux"))
}

func TestBinaryOverride(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(t *testing.T) {
			ctx := context.New(
				config.Project{
					Dist:        dist,
					ProjectName: "foobar",
					Archives: []config.Archive{
						{
							Builds:       []string{"default"},
							NameTemplate: defaultNameTemplate,
							Files: []config.File{
								{Source: "README.*"},
							},
							FormatOverrides: []config.FormatOverride{
								{
									Goos:   "windows",
									Format: "binary",
								},
							},
						},
					},
				},
			)
			ctx.Git.CurrentTag = "v0.0.1"
			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "darwin",
				Goarch: "amd64",
				Name:   "mybin",
				Path:   filepath.Join(dist, "darwinamd64", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "mybin",
					"ID":     "default",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "windows",
				Goarch: "amd64",
				Name:   "mybin.exe",
				Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "mybin",
					"Ext":    ".exe",
					"ID":     "default",
				},
			})
			ctx.Version = "0.0.1"
			ctx.Config.Archives[0].Format = format

			require.NoError(t, Pipe{}.Run(ctx))
			archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive))
			darwin := archives.Filter(artifact.ByGoos("darwin")).List()[0]
			require.Equal(t, "foobar_0.0.1_darwin_amd64."+format, darwin.Name)
			require.Equal(t, format, darwin.ExtraOr("Format", ""))
			require.Empty(t, darwin.ExtraOr("WrappedIn", ""))

			archives = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
			windows := archives.Filter(artifact.ByGoos("windows")).List()[0]
			require.Equal(t, "foobar_0.0.1_windows_amd64.exe", windows.Name)
			require.Empty(t, windows.ExtraOr("WrappedIn", ""))
		})
	}
}

func TestRunPipeSameArchiveFilename(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(
		config.Project{
			Dist:        dist,
			ProjectName: "foobar",
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "same-filename",
					Files: []config.File{
						{Source: "README.*"},
						{Source: "./foo/**/*"},
					},
					Format: "tar.gz",
				},
			},
		},
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary":    "mybin",
			"Extension": ".exe",
			"ID":        "default",
		},
	})
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	err = Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "same-filename.tar.gz already exists. Check your archive name template")
}

func TestDuplicateFilesInsideArchive(t *testing.T) {
	folder := t.TempDir()

	f, err := ioutil.TempFile(folder, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})

	ff, err := ioutil.TempFile(folder, "")
	require.NoError(t, err)
	require.NoError(t, ff.Close())

	a := NewEnhancedArchive(archive.New(f), "")
	t.Cleanup(func() {
		require.NoError(t, a.Close())
	})

	require.NoError(t, a.Add(config.File{
		Source:      ff.Name(),
		Destination: "foo",
	}))
	require.EqualError(t, a.Add(config.File{
		Source:      ff.Name(),
		Destination: "foo",
	}), "file foo already exists in the archive")
}

func TestWrapInDirectory(t *testing.T) {
	t.Run("false", func(t *testing.T) {
		require.Equal(t, "", wrapFolder(config.Archive{
			WrapInDirectory: "false",
		}))
	})
	t.Run("true", func(t *testing.T) {
		require.Equal(t, "foo", wrapFolder(config.Archive{
			WrapInDirectory: "true",
			NameTemplate:    "foo",
		}))
	})
	t.Run("custom", func(t *testing.T) {
		require.Equal(t, "foobar", wrapFolder(config.Archive{
			WrapInDirectory: "foobar",
		}))
	})
}

func TestSeveralArchivesWithTheSameID(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					ID: "a",
				},
				{
					ID: "a",
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 archives with the ID 'a', please fix your config")
}

func TestFindFiles(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tmpl := tmpl.New(context.New(config.Project{}))

	t.Run("single file", func(t *testing.T) {
		result, err := findFiles(tmpl, []config.File{
			{
				Source:      "./testdata/**/d.txt",
				Destination: "var/foobar/d.txt",
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "var/foobar/d.txt/testdata/a/b/c/d.txt",
			},
		}, result)
	})

	t.Run("match multiple files within tree without destination", func(t *testing.T) {
		result, err := findFiles(tmpl, []config.File{{Source: "./testdata/a"}})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{Source: "testdata/a/a.txt", Destination: "testdata/a/a.txt"},
			{Source: "testdata/a/b/a.txt", Destination: "testdata/a/b/a.txt"},
			{Source: "testdata/a/b/c/d.txt", Destination: "testdata/a/b/c/d.txt"},
		}, result)
	})

	t.Run("match multiple files within tree specific destination", func(t *testing.T) {
		result, err := findFiles(tmpl, []config.File{
			{
				Source:      "./testdata/a",
				Destination: "usr/local/test",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "usr/local/test/testdata/a/a.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
			{
				Source:      "testdata/a/b/a.txt",
				Destination: "usr/local/test/testdata/a/b/a.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "usr/local/test/testdata/a/b/c/d.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
		}, result)
	})

	t.Run("match multiple files within tree specific destination stripping parents", func(t *testing.T) {
		result, err := findFiles(tmpl, []config.File{
			{
				Source:      "./testdata/a",
				Destination: "usr/local/test",
				StripParent: true,
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "usr/local/test/a.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
			{
				Source:      "testdata/a/b/c/d.txt",
				Destination: "usr/local/test/d.txt",
				Info: config.FileInfo{
					Owner: "carlos",
					Group: "users",
					Mode:  0755,
					MTime: now,
				},
			},
		}, result)
	})
}

func TestArchive_globbing(t *testing.T) {
	assertGlob := func(t *testing.T, files []config.File, expected []string) {
		t.Helper()
		bin, err := ioutil.TempFile(t.TempDir(), "binary")
		require.NoError(t, err)
		dist := t.TempDir()
		ctx := context.New(config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					Format:       "tar.gz",
					NameTemplate: "foo",
					Files:        files,
				},
			},
		})

		ctx.Artifacts.Add(&artifact.Artifact{
			Goos:   "darwin",
			Goarch: "amd64",
			Name:   "foobin",
			Path:   bin.Name(),
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"ID": "default",
			},
		})

		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, append(expected, "foobin"), tarFiles(t, filepath.Join(dist, "foo.tar.gz")))
	}

	t.Run("exact src file", func(t *testing.T) {
		assertGlob(t, []config.File{{Source: "testdata/a/a.txt"}}, []string{"testdata/a/a.txt"})
	})

	t.Run("exact src file with dst", func(t *testing.T) {
		assertGlob(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "foo/",
			},
		}, []string{"foo/testdata/a/a.txt"})
	})

	t.Run("glob src", func(t *testing.T) {
		assertGlob(t, []config.File{
			{Source: "testdata/**/*.txt"},
		}, []string{
			"testdata/a/a.txt",
			"testdata/a/b/a.txt",
			"testdata/a/b/c/d.txt",
		})
	})

	t.Run("glob src with dst", func(t *testing.T) {
		assertGlob(t, []config.File{
			{
				Source:      "testdata/**/*.txt",
				Destination: "var/yada",
			},
		}, []string{
			"var/yada/testdata/a/a.txt",
			"var/yada/testdata/a/b/a.txt",
			"var/yada/testdata/a/b/c/d.txt",
		})
	})

	t.Run("glob src with dst stripping parent", func(t *testing.T) {
		assertGlob(t, []config.File{
			{
				Source:      "testdata/**/*.txt",
				Destination: "var/yada",
				StripParent: true,
			},
		}, []string{
			"var/yada/a.txt",
			"var/yada/d.txt",
		})
	})
}
