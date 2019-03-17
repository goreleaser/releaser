package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")
	repo, err := remoteRepo()
	assert.NoError(t, err)
	assert.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestExtractRepoFromURL(t *testing.T) {
	for _, url := range []string{
		"git@github.com:goreleaser/goreleaser.git",
		"git@custom:goreleaser/goreleaser.git",
		"git@custom:crazy/url/goreleaser/goreleaser.git",
		"https://github.com/goreleaser/goreleaser.git",
		"https://github.enterprise.com/goreleaser/goreleaser.git",
		"https://github.enterprise.com/crazy/url/goreleaser/goreleaser.git",
	} {
		t.Run(url, func(t *testing.T) {
			repo := extractRepoFromURL(url)
			assert.Equal(t, "goreleaser/goreleaser", repo.String())
		})
	}
}

func TestExtractRepoFromNonURL(t *testing.T) {
	for _, test := range []struct {
		expected string
		url      string
	}{
		{"", ""},
		{"repo", "git@host:repo"},
		{"repo", "git@host:repo.git"},
		{"tmp/repo2", "file:///tmp/repo2.git"},
	} {
		t.Run(test.url, func(t *testing.T) {
			repo := extractRepoFromURL(test.url)
			assert.Equal(t, test.expected, repo.String())
		})
	}
}
