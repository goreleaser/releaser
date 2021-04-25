// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket
package scoop

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNoWindows when there is no build for windows (goos doesn't contain windows).
var ErrNoWindows = errors.New("scoop requires a windows build")

// ErrTokenTypeNotImplementedForScoop indicates that a new token type was not implemented for this pipe.
var ErrTokenTypeNotImplementedForScoop = errors.New("token type not implemented for scoop pipe")

// Pipe for build.
type Pipe struct{}

func (Pipe) String() string {
	return "scoop manifests"
}

// Publish scoop manifest.
func (Pipe) Publish(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}

	client, err := client.New(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, client)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Scoop.Name == "" {
		ctx.Config.Scoop.Name = ctx.Config.ProjectName
	}
	if ctx.Config.Scoop.CommitAuthor.Name == "" {
		ctx.Config.Scoop.CommitAuthor.Name = "goreleaserbot"
	}
	if ctx.Config.Scoop.CommitAuthor.Email == "" {
		ctx.Config.Scoop.CommitAuthor.Email = "goreleaser@carlosbecker.com"
	}

	if ctx.Config.Scoop.CommitMessageTemplate == "" {
		ctx.Config.Scoop.CommitMessageTemplate = "Scoop update for {{ .ProjectName }} version {{ .Tag }}"
	}

	return nil
}

func doRun(ctx *context.Context, cl client.Client) error {
	scoop := ctx.Config.Scoop
	if scoop.Bucket.Name == "" {
		return pipe.ErrSkipDisabledPipe
	}

	if scoop.Bucket.Token != "" {
		token, err := tmpl.New(ctx).ApplySingleEnvOnly(scoop.Bucket.Token)
		if err != nil {
			return err
		}
		log.Debug("using custom token to publish scoop manifest")
		c, err := client.NewWithToken(ctx, token)
		if err != nil {
			return err
		}
		cl = c
	}

	// TODO: multiple archives
	if ctx.Config.Archives[0].Format == "binary" {
		return pipe.Skip("archive format is binary")
	}

	archives := ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("windows"),
			artifact.ByType(artifact.UploadableArchive),
		),
	).List()
	if len(archives) == 0 {
		return ErrNoWindows
	}

	path := scoop.Name + ".json"

	data, err := dataFor(ctx, cl, archives)
	if err != nil {
		return err
	}
	content, err := doBuildManifest(data)
	if err != nil {
		return err
	}

	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if strings.TrimSpace(scoop.SkipUpload) == "true" {
		return pipe.Skip("scoop.skip_upload is true")
	}
	if strings.TrimSpace(scoop.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("release is prerelease")
	}
	if ctx.Config.Release.Draft {
		return pipe.Skip("release is marked as draft")
	}
	if ctx.Config.Release.Disable {
		return pipe.Skip("release is disabled")
	}

	commitMessage, err := tmpl.New(ctx).
		Apply(scoop.CommitMessageTemplate)
	if err != nil {
		return err
	}

	repo := client.RepoFromRef(scoop.Bucket)
	return cl.CreateFile(
		ctx,
		scoop.CommitAuthor,
		repo,
		content.Bytes(),
		path,
		commitMessage,
	)
}

// Manifest represents a scoop.sh App Manifest.
// more info: https://github.com/lukesampson/scoop/wiki/App-Manifests
type Manifest struct {
	Version      string              `json:"version"`                // The version of the app that this manifest installs.
	Architecture map[string]Resource `json:"architecture"`           // `architecture`: If the app has 32- and 64-bit versions, architecture can be used to wrap the differences.
	Homepage     string              `json:"homepage,omitempty"`     // `homepage`: The home page for the program.
	License      string              `json:"license,omitempty"`      // `license`: The software license for the program. For well-known licenses, this will be a string like "MIT" or "GPL2". For custom licenses, this should be the URL of the license.
	Description  string              `json:"description,omitempty"`  // Description of the app
	Persist      []string            `json:"persist,omitempty"`      // Persist data between updates
	PreInstall   []string            `json:"pre_install,omitempty"`  // An array of strings, of the commands to be executed before an application is installed.
	PostInstall  []string            `json:"post_install,omitempty"` // An array of strings, of the commands to be executed after an application is installed.
}

// Resource represents a combination of a url and a binary name for an architecture.
type Resource struct {
	URL  string   `json:"url"`  // URL to the archive
	Bin  []string `json:"bin"`  // name of binary inside the archive
	Hash string   `json:"hash"` // the archive checksum
}

func doBuildManifest(manifest Manifest) (bytes.Buffer, error) {
	var result bytes.Buffer
	data, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return result, err
	}
	_, err = result.Write(data)
	return result, err
}

func dataFor(ctx *context.Context, cl client.Client, artifacts []*artifact.Artifact) (Manifest, error) {
	manifest := Manifest{
		Version:      ctx.Version,
		Architecture: map[string]Resource{},
		Homepage:     ctx.Config.Scoop.Homepage,
		License:      ctx.Config.Scoop.License,
		Description:  ctx.Config.Scoop.Description,
		Persist:      ctx.Config.Scoop.Persist,
		PreInstall:   ctx.Config.Scoop.PreInstall,
		PostInstall:  ctx.Config.Scoop.PostInstall,
	}

	if ctx.Config.Scoop.URLTemplate == "" {
		url, err := cl.ReleaseURLTemplate(ctx)
		if err != nil {
			if client.IsNotImplementedErr(err) {
				return manifest, ErrTokenTypeNotImplementedForScoop
			}
			return manifest, err
		}
		ctx.Config.Scoop.URLTemplate = url
	}

	for _, artifact := range artifacts {
		if artifact.Goos != "windows" {
			continue
		}

		var arch string
		switch {
		case artifact.Goarch == "386":
			arch = "32bit"
		case artifact.Goarch == "amd64":
			arch = "64bit"
		default:
			continue
		}

		url, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(ctx.Config.Scoop.URLTemplate)
		if err != nil {
			return manifest, err
		}

		sum, err := artifact.Checksum("sha256")
		if err != nil {
			return manifest, err
		}

		log.WithFields(log.Fields{
			"artifactExtras":   artifact.Extra,
			"fromURLTemplate":  ctx.Config.Scoop.URLTemplate,
			"templatedBrewURL": url,
			"sum":              sum,
		}).Debug("scoop url templating")

		manifest.Architecture[arch] = Resource{
			URL:  url,
			Bin:  binaries(artifact),
			Hash: sum,
		}
	}

	return manifest, nil
}

func binaries(a *artifact.Artifact) []string {
	// nolint: prealloc
	var bins []string
	wrap := a.ExtraOr("WrappedIn", "").(string)
	for _, b := range a.ExtraOr("Builds", []*artifact.Artifact{}).([]*artifact.Artifact) {
		bins = append(bins, filepath.ToSlash(filepath.Join(wrap, b.Name)))
	}
	return bins
}
