// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/fileglob"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/archive"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultNameTemplate       = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}"
	defaultBinaryNameTemplate = "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}"
)

// ErrArchiveDifferentBinaryCount happens when an archive uses several builds which have different goos/goarch/etc sets,
// causing the archives for some platforms to have more binaries than others.
// GoReleaser breaks in these cases as it will only cause confusion to other users.
var ErrArchiveDifferentBinaryCount = errors.New("archive has different count of built binaries for each platform, which may cause your users confusion. Please make sure all builds used have the same set of goos/goarch/etc or split it into multiple archives")

// nolint: gochecknoglobals
var lock sync.Mutex

// Pipe for archive.
type Pipe struct{}

func (Pipe) String() string {
	return "archives"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("archives")
	if len(ctx.Config.Archives) == 0 {
		ctx.Config.Archives = append(ctx.Config.Archives, config.Archive{})
	}
	for i := range ctx.Config.Archives {
		archive := &ctx.Config.Archives[i]
		if archive.Format == "" {
			archive.Format = "tar.gz"
		}
		if archive.ID == "" {
			archive.ID = "default"
		}
		if len(archive.Files) == 0 {
			archive.Files = []config.File{
				{Source: "license*"},
				{Source: "LICENSE*"},
				{Source: "readme*"},
				{Source: "README*"},
				{Source: "changelog*"},
				{Source: "CHANGELOG*"},
			}
		}
		if archive.NameTemplate == "" {
			archive.NameTemplate = defaultNameTemplate
			if archive.Format == "binary" {
				archive.NameTemplate = defaultBinaryNameTemplate
			}
		}
		if len(archive.Builds) == 0 {
			for _, build := range ctx.Config.Builds {
				archive.Builds = append(archive.Builds, build.ID)
			}
		}
		ids.Inc(archive.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.New(ctx.Parallelism)
	for i, archive := range ctx.Config.Archives {
		archive := archive
		artifacts := ctx.Artifacts.Filter(
			artifact.And(
				artifact.ByType(artifact.Binary),
				artifact.ByIDs(archive.Builds...),
			),
		).GroupByPlatform()
		if err := checkArtifacts(artifacts); err != nil && !archive.AllowDifferentBinaryCount {
			return fmt.Errorf("invalid archive: %d: %w", i, ErrArchiveDifferentBinaryCount)
		}
		for group, artifacts := range artifacts {
			log.Debugf("group %s has %d binaries", group, len(artifacts))
			artifacts := artifacts
			g.Go(func() error {
				if packageFormat(archive, artifacts[0].Goos) == "binary" {
					return skip(ctx, archive, artifacts)
				}
				return create(ctx, archive, artifacts)
			})
		}
	}
	return g.Wait()
}

func checkArtifacts(artifacts map[string][]*artifact.Artifact) error {
	lens := map[int]bool{}
	for _, v := range artifacts {
		lens[len(v)] = true
	}
	if len(lens) <= 1 {
		return nil
	}
	return ErrArchiveDifferentBinaryCount
}

func create(ctx *context.Context, arch config.Archive, binaries []*artifact.Artifact) error {
	format := packageFormat(arch, binaries[0].Goos)
	folder, err := tmpl.New(ctx).
		WithArtifact(binaries[0], arch.Replacements).
		Apply(arch.NameTemplate)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(ctx.Config.Dist, folder+"."+format)
	lock.Lock()
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755|os.ModeDir); err != nil {
		lock.Unlock()
		return err
	}
	if _, err = os.Stat(archivePath); !os.IsNotExist(err) {
		lock.Unlock()
		return fmt.Errorf("archive named %s already exists. Check your archive name template", archivePath)
	}
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("failed to create directory %s: %w", archivePath, err)
	}
	lock.Unlock()
	defer archiveFile.Close()

	log := log.WithField("archive", archivePath)
	log.Info("creating")

	template := tmpl.New(ctx).
		WithArtifact(binaries[0], arch.Replacements)
	wrap, err := template.Apply(wrapFolder(arch))
	if err != nil {
		return err
	}

	a := NewEnhancedArchive(archive.New(archiveFile), wrap)
	defer a.Close()

	files, err := findFiles(template, arch.Files)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %w", err)
	}
	for _, f := range files {
		if err = a.Add(f); err != nil {
			return fmt.Errorf("failed to add: '%s' -> '%s': %w", f.Source, f.Destination, err)
		}
	}
	for _, binary := range binaries {
		if err := a.Add(config.File{
			Source:      binary.Path,
			Destination: binary.Name,
		}); err != nil {
			return fmt.Errorf("failed to add: '%s' -> '%s': %w", binary.Path, binary.Name, err)
		}
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   folder + "." + format,
		Path:   archivePath,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
		Gomips: binaries[0].Gomips,
		Extra: map[string]interface{}{
			"Builds":    binaries,
			"ID":        arch.ID,
			"Format":    arch.Format,
			"WrappedIn": wrap,
		},
	})
	return nil
}

func wrapFolder(a config.Archive) string {
	switch a.WrapInDirectory {
	case "true":
		return a.NameTemplate
	case "false":
		return ""
	default:
		return a.WrapInDirectory
	}
}

func skip(ctx *context.Context, archive config.Archive, binaries []*artifact.Artifact) error {
	for _, binary := range binaries {
		log.WithField("binary", binary.Name).Info("skip archiving")
		name, err := tmpl.New(ctx).
			WithArtifact(binary, archive.Replacements).
			Apply(archive.NameTemplate)
		if err != nil {
			return err
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Type:   artifact.UploadableBinary,
			Name:   name + binary.ExtraOr("Ext", "").(string),
			Path:   binary.Path,
			Goos:   binary.Goos,
			Goarch: binary.Goarch,
			Goarm:  binary.Goarm,
			Gomips: binary.Gomips,
			Extra: map[string]interface{}{
				"Builds": []*artifact.Artifact{binary},
				"ID":     archive.ID,
				"Format": archive.Format,
			},
		})
	}
	return nil
}

func findFiles(template *tmpl.Template, files []config.File) ([]config.File, error) {
	var result []config.File
	for _, f := range files {
		replaced, err := template.Apply(f.Source)
		if err != nil {
			return result, fmt.Errorf("failed to apply template %s: %w", f.Source, err)
		}

		files, err := fileglob.Glob(replaced)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %w", f.Source, err)
		}

		for _, file := range files {
			result = append(result, config.File{
				Source:      file,
				Destination: destinationFor(f, file),
				Info:        f.Info,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Destination < result[j].Destination
	})

	return unique(result), nil
}

// remove duplicates
func unique(in []config.File) []config.File {
	var result []config.File
	exist := map[string]string{}
	for _, f := range in {
		if current := exist[f.Destination]; current != "" {
			log.Warnf(
				"file '%s' already exists in archive as '%s' - '%s' will be ignored",
				f.Destination,
				current,
				f.Source,
			)
			continue
		}
		exist[f.Destination] = f.Source
		result = append(result, f)
	}

	return result
}

func destinationFor(f config.File, path string) string {
	if f.Destination == "" {
		return path
	}
	if f.StripParent {
		return filepath.Join(f.Destination, filepath.Base(path))
	}
	return filepath.Join(f.Destination, path)
}

func packageFormat(archive config.Archive, platform string) string {
	for _, override := range archive.FormatOverrides {
		if strings.HasPrefix(platform, override.Goos) {
			return override.Format
		}
	}
	return archive.Format
}

// NewEnhancedArchive enhances a pre-existing archive.Archive instance
// with this pipe specifics.
func NewEnhancedArchive(a archive.Archive, wrap string) archive.Archive {
	return EnhancedArchive{
		a:     a,
		wrap:  wrap,
		files: map[string]string{},
	}
}

// EnhancedArchive is an archive.Archive implementation which decorates an
// archive.Archive adding wrap directory support, logging and windows
// backslash fixes.
type EnhancedArchive struct {
	a     archive.Archive
	wrap  string
	files map[string]string
}

// Add adds a file.
func (d EnhancedArchive) Add(f config.File) error {
	name := strings.ReplaceAll(filepath.Join(d.wrap, f.Destination), "\\", "/")
	log.Debugf("adding file: %s as %s", f.Source, name)
	if _, ok := d.files[f.Destination]; ok {
		return fmt.Errorf("file %s already exists in the archive", f.Destination)
	}
	d.files[f.Destination] = name
	ff := config.File{
		Source:      f.Source,
		Destination: name,
		Info:        f.Info,
	}
	return d.a.Add(ff)
}

// Close closes the underlying archive.
func (d EnhancedArchive) Close() error {
	return d.a.Close()
}
