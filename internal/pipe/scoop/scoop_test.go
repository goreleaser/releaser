package scoop

import (
	ctx "context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)

	var ctx = &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			ProjectName: "barr",
			Builds: []config.Build{
				{
					Binary: "foo",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
				},
				{
					Binary: "bar",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
					Ignore: []config.IgnoredBuild{
						{Goos: "darwin", Goarch: "amd64"},
					},
				},
				{
					Binary: "foobar",
					Goos:   []string{"linux"},
					Goarch: []string{"amd64"},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Scoop.Name)
	require.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Scoop.CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Scoop.CommitMessageTemplate)
}

func Test_doRun(t *testing.T) {
	var folder = testlib.Mktmp(t)
	var file = filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0644))

	type errChecker func(*testing.T, error)
	var shouldErr = func(msg string) errChecker {
		return func(t *testing.T, err error) {
			t.Helper()
			require.Error(t, err)
			require.EqualError(t, err, msg)
		}
	}
	var shouldNotErr = func(t *testing.T, err error) {
		t.Helper()
		require.NoError(t, err)
	}
	type args struct {
		ctx    *context.Context
		client client.Client
	}
	tests := []struct {
		name        string
		args        args
		artifacts   []*artifact.Artifact
		assertError errChecker
	}{
		{
			"valid public github",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
		},
		{
			"wrap in directory",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz", WrapInDirectory: "true"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"Wrap": "foo_1.0.1_windows_amd64",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"Wrap": "foo_1.0.1_windows_386",
					},
				},
			},
			shouldNotErr,
		},
		{
			"valid enterprise github",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						GitHubURLs: config.GitHubURLs{Download: "https://api.custom.github.enterprise.com"},
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldNotErr,
		},
		{
			"valid public gitlab",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitLab,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitLab: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://gitlab.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
					},
				},
			},
			shouldNotErr,
		},
		{
			"valid enterprise gitlab",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitLab,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						GitHubURLs: config.GitHubURLs{Download: "https://api.custom.gitlab.enterprise.com"},
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://gitlab.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
					},
				},
			},
			shouldNotErr,
		},
		{
			"token type not implemented for pipe",
			args{
				&context.Context{
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{NotImplemented: true},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr(ErrTokenTypeNotImplementedForScoop.Error()),
		},
		{
			"no windows build",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test"},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_linux_amd64.tar.gz", Goos: "linux", Goarch: "amd64"},
				{Name: "foo_1.0.1_linux_386.tar.gz", Goos: "linux", Goarch: "386"},
			},
			shouldErr("scoop requires a windows build"),
		},
		{
			"no scoop",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64"},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386"},
			},
			shouldErr(pipe.ErrSkipDisabledPipe.Error()),
		},
		{
			"no publish",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
					SkipPublish: true,
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr(pipe.ErrSkipPublishEnabled.Error()),
		},
		{
			"is draft",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							Draft: true,
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("release is marked as draft"),
		},
		{
			"is prerelease and skip upload set to auto",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1-pre.1",
					},
					Semver: context.Semver{
						Major:      1,
						Minor:      0,
						Patch:      1,
						Prerelease: "-pre.1",
					},
					Version:   "1.0.1-pre.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							SkipUpload: "auto",
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1-pre.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("release is prerelease"),
		},
		{
			"skip upload set to true",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							GitHub: config.Repo{
								Owner: "test",
								Name:  "test",
							},
						},
						Scoop: config.Scoop{
							SkipUpload: "true",
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1-pre.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1-pre.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("scoop.skip_upload is true"),
		},
		{
			"release is disabled",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "tar.gz"},
						},
						Release: config.Release{
							Disable: true,
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("release is disabled"),
		},
		{
			"no archive",
			args{
				&context.Context{
					TokenType: context.TokenTypeGitHub,
					Git: context.GitInfo{
						CurrentTag: "v1.0.1",
					},
					Version:   "1.0.1",
					Artifacts: artifact.New(),
					Config: config.Project{
						Builds: []config.Build{
							{Binary: "test", Goarch: []string{"amd64"}, Goos: []string{"windows"}},
						},
						Dist:        ".",
						ProjectName: "run-pipe",
						Archives: []config.Archive{
							{Format: "binary"},
						},
						Release: config.Release{
							Draft: true,
						},
						Scoop: config.Scoop{
							Bucket: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Description: "A run pipe test formula",
							Homepage:    "https://github.com/goreleaser",
						},
					},
				},
				&DummyClient{},
			},
			[]*artifact.Artifact{
				{Name: "foo_1.0.1_windows_amd64.tar.gz", Goos: "windows", Goarch: "amd64", Path: file},
				{Name: "foo_1.0.1_windows_386.tar.gz", Goos: "windows", Goarch: "386", Path: file},
			},
			shouldErr("archive format is binary"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx = tt.args.ctx
			for _, a := range tt.artifacts {
				ctx.Artifacts.Add(a)
			}
			require.NoError(t, Pipe{}.Default(ctx))

			tt.assertError(t, doRun(ctx, tt.args.client))
		})
	}
}

func Test_buildManifest(t *testing.T) {
	var folder = t.TempDir()
	var file = filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0644))

	tests := []struct {
		filename string
		ctx      *context.Context
	}{
		{
			filepath.Join("testdata", "test_buildmanifest.json.golden"),
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description: "A run pipe test formula",
						Homepage:    "https://github.com/goreleaser",
						Persist:     []string{"data", "config", "test.ini"},
					},
				},
			},
		},
		{
			filepath.Join("testdata", "test_buildmanifest_pre_post_install.json.golden"),
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description: "A run pipe test formula",
						Homepage:    "https://github.com/goreleaser",
						Persist:     []string{"data", "config", "test.ini"},
						PreInstall:  []string{"Write-Host 'Running preinstall command'"},
						PostInstall: []string{"Write-Host 'Running postinstall command'"},
					},
				},
			},
		},
		{
			filepath.Join("testdata", "test_buildmanifest_url_template.json.golden"),
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Builds: []config.Build{
						{Binary: "test"},
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description:           "A run pipe test formula",
						Homepage:              "https://github.com/goreleaser",
						URLTemplate:           "http://github.mycompany.com/foo/bar/{{ .Tag }}/{{ .ArtifactName }}",
						CommitMessageTemplate: "chore(scoop): update {{ .ProjectName }} version {{ .Tag }}",
						Persist:               []string{"data.cfg", "etc"},
					},
				},
			},
		},
		{
			filepath.Join("testdata", "test_buildmanifest_gitlab_url_template.json.golden"),
			&context.Context{
				Context:   ctx.Background(),
				TokenType: context.TokenTypeGitLab,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Config: config.Project{
					GitLabURLs: config.GitLabURLs{
						Download: "https://gitlab.com",
					},
					Builds: []config.Build{
						{Binary: "test"},
					},
					Dist:        ".",
					ProjectName: "run-pipe",
					Archives: []config.Archive{
						{Format: "tar.gz"},
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Scoop: config.Scoop{
						Bucket: config.RepoRef{
							Owner: "test",
							Name:  "test",
						},
						Description:           "A run pipe test formula",
						Homepage:              "https://gitlab.com/goreleaser",
						URLTemplate:           "http://gitlab.mycompany.com/foo/bar/uploads/{{ .ArtifactUploadHash }}/{{ .ArtifactName }}",
						CommitMessageTemplate: "chore(scoop): update {{ .ProjectName }} version {{ .Tag }}",
						Persist:               []string{"data.cfg", "etc"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			var ctx = tt.ctx
			err := Pipe{}.Default(ctx)
			require.NoError(t, err)

			cl, err := client.New(ctx)
			require.NoError(t, err)

			mf, err := dataFor(ctx, cl, []*artifact.Artifact{
				{
					Name:   "foo_1.0.1_windows_amd64.tar.gz",
					Goos:   "windows",
					Goarch: "amd64",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
						"Builds": []*artifact.Artifact{
							{
								Name: "foo.exe",
							},
							{
								Name: "bar.exe",
							},
						},
					},
				},
				{
					Name:   "foo_1.0.1_windows_arm.tar.gz",
					Goos:   "windows",
					Goarch: "arm",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
						"Builds": []*artifact.Artifact{
							{
								Name: "foo.exe",
							},
							{
								Name: "bar.exe",
							},
						},
					},
				},
				{
					Name:   "foo_1.0.1_windows_386.tar.gz",
					Goos:   "windows",
					Goarch: "386",
					Path:   file,
					Extra: map[string]interface{}{
						"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
						"Builds": []*artifact.Artifact{
							{
								Name: "foo.exe",
							},
							{
								Name: "bar.exe",
							},
						},
					},
				},
			})
			require.NoError(t, err)

			out, err := doBuildManifest(mf)
			require.NoError(t, err)

			if *update {
				require.NoError(t, os.WriteFile(tt.filename, out.Bytes(), 0655))
			}
			bts, err := os.ReadFile(tt.filename)
			require.NoError(t, err)
			require.Equal(t, string(bts), out.String())
		})
	}
}

func TestWrapInDirectory(t *testing.T) {
	var folder = t.TempDir()
	var file = filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0644))
	var ctx = &context.Context{
		TokenType: context.TokenTypeGitLab,
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Config: config.Project{
			GitLabURLs: config.GitLabURLs{
				Download: "https://gitlab.com",
			},
			Builds: []config.Build{
				{Binary: "test"},
			},
			Dist:        ".",
			ProjectName: "run-pipe",
			Archives: []config.Archive{
				{Format: "tar.gz", WrapInDirectory: "true"},
			},
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
			Scoop: config.Scoop{
				Bucket: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
				Description:           "A run pipe test formula",
				Homepage:              "https://gitlab.com/goreleaser",
				URLTemplate:           "http://gitlab.mycompany.com/foo/bar/uploads/{{ .ArtifactUploadHash }}/{{ .ArtifactName }}",
				CommitMessageTemplate: "chore(scoop): update {{ .ProjectName }} version {{ .Tag }}",
				Persist:               []string{"data.cfg", "etc"},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	cl, err := client.New(ctx)
	require.NoError(t, err)
	mf, err := dataFor(ctx, cl, []*artifact.Artifact{
		{
			Name:   "foo_1.0.1_windows_amd64.tar.gz",
			Goos:   "windows",
			Goarch: "amd64",
			Path:   file,
			Extra: map[string]interface{}{
				"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
				"WrappedIn":          "foo_1.0.1_windows_amd64",
				"Builds": []*artifact.Artifact{
					{
						Name: "foo.exe",
					},
					{
						Name: "bar.exe",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	out, err := doBuildManifest(mf)
	require.NoError(t, err)

	var golden = filepath.Join("testdata", "test_buildmanifest_wrap.json.golden")
	if *update {
		require.NoError(t, os.WriteFile(golden, out.Bytes(), 0655))
	}
	bts, err := os.ReadFile(golden)
	require.NoError(t, err)
	require.Equal(t, string(bts), out.String())
}

type DummyClient struct {
	CreatedFile    bool
	Content        string
	NotImplemented bool
}

func (dc *DummyClient) CloseMilestone(ctx *context.Context, repo client.Repo, title string) error {
	return nil
}

func (dc *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return
}

func (dc *DummyClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	if dc.NotImplemented {
		return "", client.NotImplementedError{}
	}
	return "", nil
}

func (dc *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo client.Repo, content []byte, path, msg string) (err error) {
	dc.CreatedFile = true
	dc.Content = string(content)
	return
}

func (dc *DummyClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error) {
	return
}
