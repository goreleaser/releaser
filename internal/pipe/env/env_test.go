package env

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSetDefaultTokenFiles(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		ctx := context.New(config.Project{})
		setDefaultTokenFiles(ctx)
		require.Equal(t, "~/.config/goreleaser/github_token", ctx.Config.EnvFiles.GitHubToken)
		require.Equal(t, "~/.config/goreleaser/gitlab_token", ctx.Config.EnvFiles.GitLabToken)
		require.Equal(t, "~/.config/goreleaser/gitea_token", ctx.Config.EnvFiles.GiteaToken)
	})
	t.Run("custom config config", func(t *testing.T) {
		cfg := "what"
		ctx := context.New(config.Project{
			EnvFiles: config.EnvFiles{
				GitHubToken: cfg,
			},
		})
		setDefaultTokenFiles(ctx)
		require.Equal(t, cfg, ctx.Config.EnvFiles.GitHubToken)
	})
}

func TestValidGithubEnv(t *testing.T) {
	require.NoError(t, os.Setenv("GITHUB_TOKEN", "asdf"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "asdf", ctx.Token)
	require.Equal(t, context.TokenTypeGitHub, ctx.TokenType)
	// so the tests do not depend on each other
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
}

func TestValidGitlabEnv(t *testing.T) {
	require.NoError(t, os.Setenv("GITLAB_TOKEN", "qwertz"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "qwertz", ctx.Token)
	require.Equal(t, context.TokenTypeGitLab, ctx.TokenType)
	// so the tests do not depend on each other
	require.NoError(t, os.Unsetenv("GITLAB_TOKEN"))
}

func TestValidGiteaEnv(t *testing.T) {
	require.NoError(t, os.Setenv("GITEA_TOKEN", "token"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "token", ctx.Token)
	require.Equal(t, context.TokenTypeGitea, ctx.TokenType)
	// so the tests do not depend on each other
	require.NoError(t, os.Unsetenv("GITEA_TOKEN"))
}

func TestInvalidEnv(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	require.NoError(t, os.Unsetenv("GITLAB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.Error(t, Pipe{}.Run(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), ErrMissingToken.Error())
}

func TestMultipleEnvTokens(t *testing.T) {
	require.NoError(t, os.Setenv("GITHUB_TOKEN", "asdf"))
	require.NoError(t, os.Setenv("GITLAB_TOKEN", "qwertz"))
	require.NoError(t, os.Setenv("GITEA_TOKEN", "token"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.Error(t, Pipe{}.Run(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), ErrMultipleTokens.Error())
	// so the tests do not depend on each other
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	require.NoError(t, os.Unsetenv("GITLAB_TOKEN"))
	require.NoError(t, os.Unsetenv("GITEA_TOKEN"))
}

func TestEmptyGithubFileEnv(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyGitlabFileEnv(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITLAB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyGiteaFileEnv(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITEA_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	require.Error(t, Pipe{}.Run(ctx))
}

func TestEmptyGithubEnvFile(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	f, err := ioutil.TempFile(t.TempDir(), "token")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0377))
	var ctx = &context.Context{
		Config: config.Project{
			EnvFiles: config.EnvFiles{
				GitHubToken: f.Name(),
			},
		},
	}
	require.EqualError(t, Pipe{}.Run(ctx), fmt.Sprintf("failed to load github token: open %s: permission denied", f.Name()))
}

func TestEmptyGitlabEnvFile(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITLAB_TOKEN"))
	f, err := ioutil.TempFile(t.TempDir(), "token")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0377))
	var ctx = &context.Context{
		Config: config.Project{
			EnvFiles: config.EnvFiles{
				GitLabToken: f.Name(),
			},
		},
	}
	require.EqualError(t, Pipe{}.Run(ctx), fmt.Sprintf("failed to load gitlab token: open %s: permission denied", f.Name()))
}

func TestEmptyGiteaEnvFile(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITEA_TOKEN"))
	f, err := ioutil.TempFile(t.TempDir(), "token")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0377))
	var ctx = &context.Context{
		Config: config.Project{
			EnvFiles: config.EnvFiles{
				GiteaToken: f.Name(),
			},
		},
	}
	require.EqualError(t, Pipe{}.Run(ctx), fmt.Sprintf("failed to load gitea token: open %s: permission denied", f.Name()))
}

func TestInvalidEnvChecksSkipped(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config:      config.Project{},
		SkipPublish: true,
	}
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestInvalidEnvReleaseDisabled(t *testing.T) {
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				Disable: true,
			},
		},
	}
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestLoadEnv(t *testing.T) {
	t.Run("env exists", func(t *testing.T) {
		var env = "SUPER_SECRET_ENV"
		require.NoError(t, os.Setenv(env, "1"))
		v, err := loadEnv(env, "nope")
		require.NoError(t, err)
		require.Equal(t, "1", v)
	})
	t.Run("env file exists", func(t *testing.T) {
		var env = "SUPER_SECRET_ENV_NOPE"
		require.NoError(t, os.Unsetenv(env))
		f, err := ioutil.TempFile(t.TempDir(), "token")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		fmt.Fprintf(f, "123")
		require.NoError(t, f.Close())
		v, err := loadEnv(env, f.Name())
		require.NoError(t, err)
		require.Equal(t, "123", v)
	})
	t.Run("env file with an empty line at the end", func(t *testing.T) {
		var env = "SUPER_SECRET_ENV_NOPE"
		require.NoError(t, os.Unsetenv(env))
		f, err := ioutil.TempFile(t.TempDir(), "token")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		fmt.Fprintf(f, "123\n")
		require.NoError(t, f.Close())
		v, err := loadEnv(env, f.Name())
		require.NoError(t, err)
		require.Equal(t, "123", v)
	})
	t.Run("env file is not readable", func(t *testing.T) {
		var env = "SUPER_SECRET_ENV_NOPE"
		require.NoError(t, os.Unsetenv(env))
		f, err := ioutil.TempFile(t.TempDir(), "token")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		fmt.Fprintf(f, "123")
		require.NoError(t, f.Close())
		err = os.Chmod(f.Name(), 0377)
		require.NoError(t, err)
		v, err := loadEnv(env, f.Name())
		require.EqualError(t, err, fmt.Sprintf("open %s: permission denied", f.Name()))
		require.Equal(t, "", v)
	})
}
