package golang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	api "github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var runtimeTarget = runtime.GOOS + "_" + runtime.GOARCH

func TestWithDefaults(t *testing.T) {
	for name, testcase := range map[string]struct {
		build    config.Build
		targets  []string
		goBinary string
	}{
		"full": {
			build: config.Build{
				ID:     "foo",
				Binary: "foo",
				Goos: []string{
					"linux",
					"windows",
					"darwin",
				},
				Goarch: []string{
					"amd64",
					"arm",
					"mips",
				},
				Goarm: []string{
					"6",
				},
				Gomips: []string{
					"softfloat",
				},
				GoBinary: "go1.2.3",
			},
			targets: []string{
				"linux_amd64",
				"linux_mips_softfloat",
				"darwin_amd64",
				"windows_amd64",
				"windows_arm_6",
				"linux_arm_6",
			},
			goBinary: "go1.2.3",
		},
		"empty": {
			build: config.Build{
				ID:     "foo2",
				Binary: "foo",
			},
			targets: []string{
				"linux_amd64",
				"linux_386",
				"linux_arm64",
				"darwin_amd64",
				"darwin_arm64",
			},
			goBinary: "go",
		},
		"custom targets": {
			build: config.Build{
				ID:     "foo3",
				Binary: "foo",
				Targets: []string{
					"linux_386",
					"darwin_amd64",
				},
			},
			targets: []string{
				"linux_386",
				"darwin_amd64",
			},
			goBinary: "go",
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := config.Project{
				Builds: []config.Build{
					testcase.build,
				},
			}
			ctx := context.New(config)
			ctx.Git.CurrentTag = "5.6.7"
			build, err := Default.WithDefaults(ctx.Config.Builds[0])
			require.NoError(t, err)
			require.ElementsMatch(t, build.Targets, testcase.targets)
			require.EqualValues(t, testcase.goBinary, build.GoBinary)
		})
	}
}

func TestInvalidTargets(t *testing.T) {
	type testcase struct {
		build       config.Build
		expectedErr string
	}
	for s, tc := range map[string]testcase{
		"goos": {
			build: config.Build{
				Goos: []string{"darwin", "darwim"},
			},
			expectedErr: "invalid goos: darwim",
		},
		"goarch": {
			build: config.Build{
				Goarch: []string{"amd64", "i386", "386"},
			},
			expectedErr: "invalid goarch: i386",
		},
		"goarm": {
			build: config.Build{
				Goarch: []string{"arm"},
				Goarm:  []string{"6", "9", "8", "7"},
			},
			expectedErr: "invalid goarm: 9",
		},
		"gomips": {
			build: config.Build{
				Goarch: []string{"mips"},
				Gomips: []string{"softfloat", "mehfloat", "hardfloat"},
			},
			expectedErr: "invalid gomips: mehfloat",
		},
	} {
		t.Run(s, func(t *testing.T) {
			config := config.Project{
				Builds: []config.Build{
					tc.build,
				},
			}
			ctx := context.New(config)
			_, err := Default.WithDefaults(ctx.Config.Builds[0])
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestBuild(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Env:    []string{"GO111MODULE=off"},
				Binary: "bin/foo-{{ .Version }}",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"windows_amd64",
					"linux_arm_6",
					"js_wasm",
					"linux_mips_softfloat",
					"linux_mips64le_softfloat",
				},
				Asmflags: []string{".=", "all="},
				Gcflags:  []string{"all="},
				Flags:    []string{"{{.Env.GO_FLAGS}}"},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Env["GO_FLAGS"] = "-v"
	ctx.Git.CurrentTag = "5.6.7"
	ctx.Version = "v" + ctx.Git.CurrentTag
	build := ctx.Config.Builds[0]
	for _, target := range build.Targets {
		var ext string
		if strings.HasPrefix(target, "windows") {
			ext = ".exe"
		} else if target == "js_wasm" {
			ext = ".wasm"
		}
		bin, terr := tmpl.New(ctx).Apply(build.Binary)
		require.NoError(t, terr)

		// injecting some delay here to force inconsistent mod times on bins
		time.Sleep(2 * time.Second)

		err := Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   bin + ext,
			Path:   filepath.Join(folder, "dist", target, bin+ext),
			Ext:    ext,
		})
		require.NoError(t, err)
	}
	require.ElementsMatch(t, ctx.Artifacts.List(), []*artifact.Artifact{
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_amd64", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_mips_softfloat", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "mips",
			Gomips: "softfloat",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_mips64le_softfloat", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "mips64le",
			Gomips: "softfloat",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "darwin_amd64", "bin", "foo-v5.6.7"),
			Goos:   "darwin",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7",
			Path:   filepath.Join(folder, "dist", "linux_arm_6", "bin", "foo-v5.6.7"),
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    "",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7.exe",
			Path:   filepath.Join(folder, "dist", "windows_amd64", "bin", "foo-v5.6.7.exe"),
			Goos:   "windows",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    ".exe",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
		{
			Name:   "bin/foo-v5.6.7.wasm",
			Path:   filepath.Join(folder, "dist", "js_wasm", "bin", "foo-v5.6.7.wasm"),
			Goos:   "js",
			Goarch: "wasm",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				"Ext":    ".wasm",
				"Binary": "foo-v5.6.7",
				"ID":     "foo",
			},
		},
	})

	modTimes := map[time.Time]bool{}
	for _, bin := range ctx.Artifacts.List() {
		if bin.Type != artifact.Binary {
			continue
		}

		fi, err := os.Stat(bin.Path)
		require.NoError(t, err)

		// make this a suitable map key, per docs: https://golang.org/pkg/time/#Time
		modTime := fi.ModTime().UTC().Round(0)

		if modTimes[modTime] {
			t.Fatal("duplicate modified time found, times should be different by default")
		}
		modTimes[modTime] = true
	}
}

func TestBuildCodeInSubdir(t *testing.T) {
	folder := testlib.Mktmp(t)
	subdir := filepath.Join(folder, "bar")
	err := os.Mkdir(subdir, 0o755)
	require.NoError(t, err)
	writeGoodMain(t, subdir)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Env:    []string{"GO111MODULE=off"},
				Dir:    "bar",
				Binary: "foo",
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	build := ctx.Config.Builds[0]
	err = Default.Build(ctx, build, api.Options{
		Target: runtimeTarget,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", runtimeTarget, build.Binary),
		Ext:    "",
	})
	require.NoError(t, err)
}

func TestBuildWithDotGoDir(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.Mkdir(filepath.Join(folder, ".go"), 0o755))
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:       "foo",
				Env:      []string{"GO111MODULE=off"},
				Binary:   "foo",
				Targets:  []string{runtimeTarget},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	build := ctx.Config.Builds[0]
	require.NoError(t, Default.Build(ctx, build, api.Options{
		Target: runtimeTarget,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", runtimeTarget, build.Binary),
		Ext:    "",
	}))
}

func TestBuildFailed(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				ID:    "buildid",
				Flags: []string{"-flag-that-dont-exists-to-force-failure"},
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: "darwin_amd64",
	})
	assertContainsError(t, err, `flag provided but not defined: -flag-that-dont-exists-to-force-failure`)
	require.Empty(t, ctx.Artifacts.List())
}

func TestBuildInvalidTarget(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	target := "linux"
	config := config.Project{
		Builds: []config.Build{
			{
				ID:      "foo",
				Binary:  "foo",
				Targets: []string{target},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	build := ctx.Config.Builds[0]
	err := Default.Build(ctx, build, api.Options{
		Target: target,
		Name:   build.Binary,
		Path:   filepath.Join(folder, "dist", target, build.Binary),
	})
	require.EqualError(t, err, "linux is not a valid build target")
	require.Len(t, ctx.Artifacts.List(), 0)
}

func TestRunInvalidAsmflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				Binary:   "nametest",
				Asmflags: []string{"{{.Version}"},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidGcflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				Binary:  "nametest",
				Gcflags: []string{"{{.Version}"},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidLdflags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				Binary:  "nametest",
				Flags:   []string{"-v"},
				Ldflags: []string{"-s -w -X main.version={{.Version}"},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunInvalidFlags(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				Binary: "nametest",
				Flags:  []string{"{{.Env.GOOS}"},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	ctx := context.New(config)
	err := Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	})
	require.EqualError(t, err, `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithoutMainFunc(t *testing.T) {
	folder := testlib.Mktmp(t)
	writeMainWithoutMainFunc(t, folder)
	config := config.Project{
		Builds: []config.Build{
			{
				Binary: "no-main",
				Hooks:  config.HookConfig{},
				Targets: []string{
					runtimeTarget,
				},
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("not main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		err:=Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		})
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
	t.Run("fixed main.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "main.go"
		require.EqualError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}), `build for no-main does not contain a main function`)
	})
}

func TestRunPipeWithProxiedRepo(t *testing.T) {
	folder := testlib.Mktmp(t)
	proxied := filepath.Join(folder, "dist/proxy/default")
	require.NoError(t, os.MkdirAll(proxied, 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(proxied, "main.go"),
		[]byte("// +build: main\npackage main\nimport github.com/goreleaser/goreleaser"),
		0o666,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(proxied, "go.mod"),
		[]byte("module foo\nrequire github.com/goreleaser/goreleaser v0.161.1"),
		0o666,
	))
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = proxied
	require.NoError(t, cmd.Run())
	config := config.Project{
		GoMod: config.GoMod{
			Proxy: true,
		},
		Builds: []config.Build{
			{
				Binary: "foo",
				Hooks:  config.HookConfig{},
				Main:   "github.com/goreleaser/goreleaser",
				Dir:    proxied,
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)

	require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Target: runtimeTarget,
	}))
}

func TestRunPipeWithMainFuncNotInMainGoFile(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "foo.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0o644,
	))
	config := config.Project{
		Builds: []config.Build{
			{
				Env:    []string{"GO111MODULE=off"},
				Binary: "foo",
				Hooks:  config.HookConfig{},
				Targets: []string{
					runtimeTarget,
				},
				GoBinary: "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Git.CurrentTag = "5.6.7"
	t.Run("empty", func(t *testing.T) {
		ctx.Config.Builds[0].Main = ""
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
	t.Run("foo.go", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "foo.go"
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
	t.Run("glob", func(t *testing.T) {
		ctx.Config.Builds[0].Main = "."
		require.NoError(t, Default.Build(ctx, ctx.Config.Builds[0], api.Options{
			Target: runtimeTarget,
		}))
	})
}

func TestLdFlagsFullTemplate(t *testing.T) {
	run := time.Now().UTC()
	commit := time.Now().AddDate(-1, 0, 0)

	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
			Commit:     "123",
			CommitDate: commit,
		},
		Date:    run,
		Version: "1.2.3",
		Env:     map[string]string{"FOO": "123"},
	}
	artifact := &artifact.Artifact{Goarch: "amd64"}
	flags, err := tmpl.New(ctx).WithArtifact(artifact, map[string]string{}).
		Apply(`-s -w -X main.version={{.Version}} -X main.tag={{.Tag}} -X main.date={{.Date}} -X main.commit={{.Commit}} -X "main.foo={{.Env.FOO}}" -X main.time={{ time "20060102" }} -X main.arch={{.Arch}} -X main.commitDate={{.CommitDate}}`)
	require.NoError(t, err)
	require.Contains(t, flags, "-s -w")
	require.Contains(t, flags, "-X main.version=1.2.3")
	require.Contains(t, flags, "-X main.tag=v1.2.3")
	require.Contains(t, flags, "-X main.commit=123")
	require.Contains(t, flags, fmt.Sprintf("-X main.date=%d", run.Year()))
	require.Contains(t, flags, fmt.Sprintf("-X main.time=%d", run.Year()))
	require.Contains(t, flags, `-X "main.foo=123"`)
	require.Contains(t, flags, `-X main.arch=amd64`)
	require.Contains(t, flags, fmt.Sprintf("-X main.commitDate=%d", commit.Year()))
}

func TestInvalidTemplate(t *testing.T) {
	for template, eerr := range map[string]string{
		"{{ .Nope }":    `template: tmpl:1: unexpected "}" in operand`,
		"{{.Env.NOPE}}": `template: tmpl:1:6: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`,
	} {
		t.Run(template, func(t *testing.T) {
			ctx := context.New(config.Project{})
			ctx.Git.CurrentTag = "3.4.1"
			flags, err := tmpl.New(ctx).Apply(template)
			require.EqualError(t, err, eerr)
			require.Empty(t, flags)
		})
	}
}

func TestProcessFlags(t *testing.T) {
	ctx := &context.Context{
		Version: "1.2.3",
	}
	ctx.Git.CurrentTag = "5.6.7"

	artifact := &artifact.Artifact{
		Name:   "name",
		Goos:   "darwin",
		Goarch: "amd64",
		Goarm:  "7",
		Extra: map[string]interface{}{
			"Binary": "binary",
		},
	}

	source := []string{
		"flag",
		"{{.Version}}",
		"{{.Os}}",
		"{{.Arch}}",
		"{{.Arm}}",
		"{{.Binary}}",
		"{{.ArtifactName}}",
	}

	expected := []string{
		"-testflag=flag",
		"-testflag=1.2.3",
		"-testflag=darwin",
		"-testflag=amd64",
		"-testflag=7",
		"-testflag=binary",
		"-testflag=name",
	}

	flags, err := processFlags(ctx, artifact, []string{}, source, "-testflag=")
	require.NoError(t, err)
	require.Len(t, flags, 7)
	require.Equal(t, expected, flags)
}

func TestProcessFlagsInvalid(t *testing.T) {
	ctx := &context.Context{}

	source := []string{
		"{{.Version}",
	}

	expected := `template: tmpl:1: unexpected "}" in operand`

	flags, err := processFlags(ctx, &artifact.Artifact{}, []string{}, source, "-testflag=")
	require.EqualError(t, err, expected)
	require.Nil(t, flags)
}

func TestJoinLdFlags(t *testing.T) {
	tests := []struct {
		input  []string
		output string
	}{
		{[]string{"-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser"}, "-ldflags=-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser"},
		{[]string{"-s -w", "-X main.version={{.Version}}"}, "-ldflags=-s -w -X main.version={{.Version}}"},
	}

	for _, test := range tests {
		joinedLdFlags := joinLdFlags(test.input)
		require.Equal(t, joinedLdFlags, test.output)
	}
}

func TestBuildModTimestamp(t *testing.T) {
	// round to seconds since this will be a unix timestamp
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()

	folder := testlib.Mktmp(t)
	writeGoodMain(t, folder)

	config := config.Project{
		Builds: []config.Build{
			{
				ID:     "foo",
				Env:    []string{"GO111MODULE=off"},
				Binary: "bin/foo-{{ .Version }}",
				Targets: []string{
					"linux_amd64",
					"darwin_amd64",
					"windows_amd64",
					"linux_arm_6",
					"js_wasm",
					"linux_mips_softfloat",
					"linux_mips64le_softfloat",
				},
				Asmflags:     []string{".=", "all="},
				Gcflags:      []string{"all="},
				Flags:        []string{"{{.Env.GO_FLAGS}}"},
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				GoBinary:     "go",
			},
		},
	}
	ctx := context.New(config)
	ctx.Env["GO_FLAGS"] = "-v"
	ctx.Git.CurrentTag = "5.6.7"
	ctx.Version = "v" + ctx.Git.CurrentTag
	build := ctx.Config.Builds[0]
	for _, target := range build.Targets {
		var ext string
		if strings.HasPrefix(target, "windows") {
			ext = ".exe"
		} else if target == "js_wasm" {
			ext = ".wasm"
		}
		bin, terr := tmpl.New(ctx).Apply(build.Binary)
		require.NoError(t, terr)

		// injecting some delay here to force inconsistent mod times on bins
		time.Sleep(2 * time.Second)

		err := Default.Build(ctx, build, api.Options{
			Target: target,
			Name:   bin + ext,
			Path:   filepath.Join(folder, "dist", target, bin+ext),
			Ext:    ext,
		})
		require.NoError(t, err)
	}

	for _, bin := range ctx.Artifacts.List() {
		if bin.Type != artifact.Binary {
			continue
		}

		fi, err := os.Stat(bin.Path)
		require.NoError(t, err)
		require.True(t, modTime.Equal(fi.ModTime()), "inconsistent mod times found when specifying ModTimestamp")
	}
}

//
// Helpers
//

func writeMainWithoutMainFunc(t *testing.T, folder string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nconst a = 2\nfunc notMain() {println(0)}"),
		0o644,
	))
}

func writeGoodMain(t *testing.T, folder string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "main.go"),
		[]byte("package main\nvar a = 1\nfunc main() {println(0)}"),
		0o644,
	))
}

func assertContainsError(t *testing.T, err error, s string) {
	t.Helper()
	require.Error(t, err)
	require.Contains(t, err.Error(), s)
}
