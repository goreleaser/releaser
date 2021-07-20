// Package config contains the model and loader of the goreleaser configuration
// file.
package config

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/goreleaser/nfpm/v2/files"
	yaml "gopkg.in/yaml.v2"
)

// GitHubURLs holds the URLs to be used when using github enterprise.
type GitHubURLs struct {
	API           string `yaml:"api,omitempty"`
	Upload        string `yaml:"upload,omitempty"`
	Download      string `yaml:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
}

// GitLabURLs holds the URLs to be used when using gitlab ce/enterprise.
type GitLabURLs struct {
	API           string `yaml:"api,omitempty"`
	Download      string `yaml:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
}

// GiteaURLs holds the URLs to be used when using gitea.
type GiteaURLs struct {
	API           string `yaml:"api,omitempty"`
	Download      string `yaml:"download,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
}

// Repo represents any kind of repo (github, gitlab, etc).
// to upload releases into.
type Repo struct {
	Owner string `yaml:",omitempty"`
	Name  string `yaml:",omitempty"`
}

// RepoRef represents any kind of repo which may differ
// from the one we are building from and may therefore
// also require separate authentication
// e.g. Homebrew Tap, Scoop bucket.
type RepoRef struct {
	Owner string `yaml:",omitempty"`
	Name  string `yaml:",omitempty"`
	Token string `yaml:",omitempty"`
}

// HomebrewDependency represents Homebrew dependency.
type HomebrewDependency struct {
	Name string `yaml:",omitempty"`
	Type string `yaml:",omitempty"`
}

// type alias to prevent stack overflowing in the custom unmarshaler.
type homebrewDependency HomebrewDependency

// UnmarshalYAML is a custom unmarshaler that accept brew deps in both the old and new format.
func (a *HomebrewDependency) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		a.Name = str
		return nil
	}

	var dep homebrewDependency
	if err := unmarshal(&dep); err != nil {
		return err
	}

	a.Name = dep.Name
	a.Type = dep.Type

	return nil
}

// String of the repo, e.g. owner/name.
func (r Repo) String() string {
	if r.Owner == "" && r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

// Homebrew contains the brew section.
type Homebrew struct {
	Name             string               `yaml:",omitempty"`
	Tap              RepoRef              `yaml:",omitempty"`
	CommitAuthor     CommitAuthor         `yaml:"commit_author,omitempty"`
	Folder           string               `yaml:",omitempty"`
	Caveats          string               `yaml:",omitempty"`
	Plist            string               `yaml:",omitempty"`
	Install          string               `yaml:",omitempty"`
	PostInstall      string               `yaml:"post_install,omitempty"`
	Dependencies     []HomebrewDependency `yaml:",omitempty"`
	Test             string               `yaml:",omitempty"`
	Conflicts        []string             `yaml:",omitempty"`
	Description      string               `yaml:",omitempty"`
	Homepage         string               `yaml:",omitempty"`
	License          string               `yaml:",omitempty"`
	SkipUpload       string               `yaml:"skip_upload,omitempty"`
	DownloadStrategy string               `yaml:"download_strategy,omitempty"`
	URLTemplate      string               `yaml:"url_template,omitempty"`
	CustomRequire    string               `yaml:"custom_require,omitempty"`
	CustomBlock      string               `yaml:"custom_block,omitempty"`
	IDs              []string             `yaml:"ids,omitempty"`
	Goarm            string               `yaml:"goarm,omitempty"`
}

// Scoop contains the scoop.sh section.
type Scoop struct {
	Name                  string       `yaml:",omitempty"`
	Bucket                RepoRef      `yaml:",omitempty"`
	Folder                string       `yaml:",omitempty"`
	CommitAuthor          CommitAuthor `yaml:"commit_author,omitempty"`
	CommitMessageTemplate string       `yaml:"commit_msg_template,omitempty"`
	Homepage              string       `yaml:",omitempty"`
	Description           string       `yaml:",omitempty"`
	License               string       `yaml:",omitempty"`
	URLTemplate           string       `yaml:"url_template,omitempty"`
	Persist               []string     `yaml:"persist,omitempty"`
	SkipUpload            string       `yaml:"skip_upload,omitempty"`
	PreInstall            []string     `yaml:"pre_install,omitempty"`
	PostInstall           []string     `yaml:"post_install,omitempty"`
}

// CommitAuthor is the author of a Git commit.
type CommitAuthor struct {
	Name  string `yaml:",omitempty"`
	Email string `yaml:",omitempty"`
}

// Hooks define actions to run before and/or after something.
type Hooks struct {
	Pre  string `yaml:",omitempty"`
	Post string `yaml:",omitempty"`
}

// IgnoredBuild represents a build ignored by the user.
type IgnoredBuild struct {
	Goos, Goarch, Goarm, Gomips string
}

// StringArray is a wrapper for an array of strings.
type StringArray []string

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strings []string
	if err := unmarshal(&strings); err != nil {
		var str string
		if err := unmarshal(&str); err != nil {
			return err
		}
		*a = []string{str}
	} else {
		*a = strings
	}
	return nil
}

// FlagArray is a wrapper for an array of strings.
type FlagArray []string

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *FlagArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var flags []string
	if err := unmarshal(&flags); err != nil {
		var flagstr string
		if err := unmarshal(&flagstr); err != nil {
			return err
		}
		*a = strings.Fields(flagstr)
	} else {
		*a = flags
	}
	return nil
}

// Build contains the build configuration section.
type Build struct {
	ID              string         `yaml:",omitempty"`
	Goos            []string       `yaml:",omitempty"`
	Goarch          []string       `yaml:",omitempty"`
	Goarm           []string       `yaml:",omitempty"`
	Gomips          []string       `yaml:",omitempty"`
	Targets         []string       `yaml:",omitempty"`
	Ignore          []IgnoredBuild `yaml:",omitempty"`
	Dir             string         `yaml:",omitempty"`
	Main            string         `yaml:",omitempty"`
	Ldflags         StringArray    `yaml:",omitempty"`
	Tags            FlagArray      `yaml:",omitempty"`
	Flags           FlagArray      `yaml:",omitempty"`
	Binary          string         `yaml:",omitempty"`
	Hooks           HookConfig     `yaml:",omitempty"`
	Env             []string       `yaml:",omitempty"`
	Lang            string         `yaml:",omitempty"`
	Asmflags        StringArray    `yaml:",omitempty"`
	Gcflags         StringArray    `yaml:",omitempty"`
	ModTimestamp    string         `yaml:"mod_timestamp,omitempty"`
	Skip            bool           `yaml:",omitempty"`
	GoBinary        string         `yaml:",omitempty"`
	NoUniqueDistDir bool           `yaml:"no_unique_dist_dir,omitempty"`
}

type HookConfig struct {
	Pre  BuildHooks `yaml:",omitempty"`
	Post BuildHooks `yaml:",omitempty"`
}

type BuildHooks []BuildHook

// UnmarshalYAML is a custom unmarshaler that allows simplified declaration of single command.
func (bhc *BuildHooks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var singleCmd string
	err := unmarshal(&singleCmd)
	if err == nil {
		*bhc = []BuildHook{{Cmd: singleCmd}}
		return nil
	}

	type t BuildHooks
	var hooks t
	if err := unmarshal(&hooks); err != nil {
		return err
	}
	*bhc = (BuildHooks)(hooks)
	return nil
}

type BuildHook struct {
	Dir string   `yaml:",omitempty"`
	Cmd string   `yaml:",omitempty"`
	Env []string `yaml:",omitempty"`
}

// UnmarshalYAML is a custom unmarshaler that allows simplified declarations of commands as strings.
func (bh *BuildHook) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string
	if err := unmarshal(&cmd); err != nil {
		type t BuildHook
		var hook t
		if err := unmarshal(&hook); err != nil {
			return err
		}
		*bh = (BuildHook)(hook)
		return nil
	}

	bh.Cmd = cmd
	return nil
}

// FormatOverride is used to specify a custom format for a specific GOOS.
type FormatOverride struct {
	Goos   string `yaml:",omitempty"`
	Format string `yaml:",omitempty"`
}

// File is a file inside an archive.
type File struct {
	Source      string   `yaml:"src,omitempty"`
	Destination string   `yaml:"dst,omitempty"`
	StripParent bool     `yaml:"strip_parent,omitempty"`
	Info        FileInfo `yaml:"info,omitempty"`
}

// FileInfo is the file info of a file.
type FileInfo struct {
	Owner string      `yaml:"owner,omitempty"`
	Group string      `yaml:"group"`
	Mode  os.FileMode `yaml:"mode,omitempty"`
	MTime time.Time   `yaml:"mtime,omitempty"`
}

// type alias to prevent stack overflow
type fileAlias File

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (f *File) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		*f = File{
			Source:      str,
			Destination: str,
		}
		return nil
	}

	var file fileAlias
	if err := unmarshal(&file); err != nil {
		return err
	}
	*f = File(file)
	return nil
}

// Archive config used for the archive.
type Archive struct {
	ID                        string            `yaml:",omitempty"`
	Builds                    []string          `yaml:",omitempty"`
	NameTemplate              string            `yaml:"name_template,omitempty"`
	Replacements              map[string]string `yaml:",omitempty"`
	Format                    string            `yaml:",omitempty"`
	FormatOverrides           []FormatOverride  `yaml:"format_overrides,omitempty"`
	WrapInDirectory           string            `yaml:"wrap_in_directory,omitempty"`
	Files                     []File            `yaml:",omitempty"`
	AllowDifferentBinaryCount bool              `yaml:"allow_different_binary_count"`
}

// Release config used for the GitHub/GitLab release.
type Release struct {
	GitHub                 Repo        `yaml:",omitempty"`
	GitLab                 Repo        `yaml:",omitempty"`
	Gitea                  Repo        `yaml:",omitempty"`
	Draft                  bool        `yaml:",omitempty"`
	Disable                bool        `yaml:",omitempty"`
	Prerelease             string      `yaml:",omitempty"`
	NameTemplate           string      `yaml:"name_template,omitempty"`
	IDs                    []string    `yaml:"ids,omitempty"`
	ExtraFiles             []ExtraFile `yaml:"extra_files,omitempty"`
	DiscussionCategoryName string      `yaml:"discussion_category_name,omitempty"`
	Header                 string      `yaml:"header,omitempty"`
	Footer                 string      `yaml:"footer,omitempty"`
}

// Milestone config used for VCS milestone.
type Milestone struct {
	Repo         Repo   `yaml:",omitempty"`
	Close        bool   `yaml:",omitempty"`
	FailOnError  bool   `yaml:"fail_on_error,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
}

// ExtraFile on a release.
type ExtraFile struct {
	Glob string `yaml:"glob,omitempty"`
}

// NFPM config.
type NFPM struct {
	NFPMOverridables `yaml:",inline"`
	Overrides        map[string]NFPMOverridables `yaml:"overrides,omitempty"`

	ID          string   `yaml:",omitempty"`
	Builds      []string `yaml:",omitempty"`
	Formats     []string `yaml:",omitempty"`
	Section     string   `yaml:",omitempty"`
	Priority    string   `yaml:",omitempty"`
	Vendor      string   `yaml:",omitempty"`
	Homepage    string   `yaml:",omitempty"`
	Maintainer  string   `yaml:",omitempty"`
	Description string   `yaml:",omitempty"`
	License     string   `yaml:",omitempty"`
	Bindir      string   `yaml:",omitempty"`
	Meta        bool     `yaml:",omitempty"` // make package without binaries - only deps
}

// NFPMScripts is used to specify maintainer scripts.
type NFPMScripts struct {
	PreInstall  string `yaml:"preinstall,omitempty"`
	PostInstall string `yaml:"postinstall,omitempty"`
	PreRemove   string `yaml:"preremove,omitempty"`
	PostRemove  string `yaml:"postremove,omitempty"`
}

type NFPMRPMSignature struct {
	// PGP secret key, can be ASCII-armored
	KeyFile       string `yaml:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-"` // populated from environment variable
}

// NFPMRPMScripts represents scripts only available on RPM packages.
type NFPMRPMScripts struct {
	PreTrans  string `yaml:"pretrans,omitempty"`
	PostTrans string `yaml:"posttrans,omitempty"`
}

// NFPMRPM is custom configs that are only available on RPM packages.
type NFPMRPM struct {
	Summary     string           `yaml:"summary,omitempty"`
	Group       string           `yaml:"group,omitempty"`
	Compression string           `yaml:"compression,omitempty"`
	Signature   NFPMRPMSignature `yaml:"signature,omitempty"`
	Scripts     NFPMRPMScripts   `yaml:"scripts,omitempty"`
}

// NFPMDebScripts is scripts only available on deb packages.
type NFPMDebScripts struct {
	Rules     string `yaml:"rules,omitempty"`
	Templates string `yaml:"templates,omitempty"`
}

// NFPMDebTriggers contains triggers only available for deb packages.
// https://wiki.debian.org/DpkgTriggers
// https://man7.org/linux/man-pages/man5/deb-triggers.5.html
type NFPMDebTriggers struct {
	Interest        []string `yaml:"interest,omitempty"`
	InterestAwait   []string `yaml:"interest_await,omitempty"`
	InterestNoAwait []string `yaml:"interest_noawait,omitempty"`
	Activate        []string `yaml:"activate,omitempty"`
	ActivateAwait   []string `yaml:"activate_await,omitempty"`
	ActivateNoAwait []string `yaml:"activate_noawait,omitempty"`
}

// NFPMDebSignature contains config for signing deb packages created by nfpm.
type NFPMDebSignature struct {
	// PGP secret key, can be ASCII-armored
	KeyFile       string `yaml:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-"` // populated from environment variable
	// origin, maint or archive (defaults to origin)
	Type string `yaml:"type,omitempty"`
}

// NFPMDeb is custom configs that are only available on deb packages.
type NFPMDeb struct {
	Scripts   NFPMDebScripts   `yaml:"scripts,omitempty"`
	Triggers  NFPMDebTriggers  `yaml:"triggers,omitempty"`
	Breaks    []string         `yaml:"breaks,omitempty"`
	Signature NFPMDebSignature `yaml:"signature,omitempty"`
}

type NFPMAPKScripts struct {
	PreUpgrade  string `yaml:"preupgrade,omitempty"`
	PostUpgrade string `yaml:"postupgrade,omitempty"`
}

// NFPMAPKSignature contains config for signing apk packages created by nfpm.
type NFPMAPKSignature struct {
	// RSA private key in PEM format
	KeyFile       string `yaml:"key_file,omitempty"`
	KeyPassphrase string `yaml:"-"` // populated from environment variable
	// defaults to <maintainer email>.rsa.pub
	KeyName string `yaml:"key_name,omitempty"`
}

// NFPMAPK is custom config only available on apk packages.
type NFPMAPK struct {
	Scripts   NFPMAPKScripts   `yaml:"scripts,omitempty"`
	Signature NFPMAPKSignature `yaml:"signature,omitempty"`
}

// NFPMOverridables is used to specify per package format settings.
type NFPMOverridables struct {
	FileNameTemplate string            `yaml:"file_name_template,omitempty"`
	PackageName      string            `yaml:"package_name,omitempty"`
	Epoch            string            `yaml:"epoch,omitempty"`
	Release          string            `yaml:"release,omitempty"`
	Prerelease       string            `yaml:"prerelease,omitempty"`
	VersionMetadata  string            `yaml:"version_metadata,omitempty"`
	Replacements     map[string]string `yaml:",omitempty"`
	Dependencies     []string          `yaml:",omitempty"`
	Recommends       []string          `yaml:",omitempty"`
	Suggests         []string          `yaml:",omitempty"`
	Conflicts        []string          `yaml:",omitempty"`
	Replaces         []string          `yaml:",omitempty"`
	EmptyFolders     []string          `yaml:"empty_folders,omitempty"`
	Contents         files.Contents    `yaml:"contents,omitempty"`
	Scripts          NFPMScripts       `yaml:"scripts,omitempty"`
	RPM              NFPMRPM           `yaml:"rpm,omitempty"`
	Deb              NFPMDeb           `yaml:"deb,omitempty"`
	APK              NFPMAPK           `yaml:"apk,omitempty"`
}

// Sign config.
type Sign struct {
	ID        string   `yaml:"id,omitempty"`
	Cmd       string   `yaml:"cmd,omitempty"`
	Args      []string `yaml:"args,omitempty"`
	Signature string   `yaml:"signature,omitempty"`
	Artifacts string   `yaml:"artifacts,omitempty"`
	IDs       []string `yaml:"ids,omitempty"`
	Stdin     *string  `yaml:"stdin,omitempty"`
	StdinFile string   `yaml:"stdin_file,omitempty"`
}

// SnapcraftAppMetadata for the binaries that will be in the snap package.
type SnapcraftAppMetadata struct {
	Plugs            []string
	Daemon           string
	Args             string
	Completer        string `yaml:",omitempty"`
	Command          string `yaml:"command"`
	RestartCondition string `yaml:"restart_condition,omitempty"`
}

type SnapcraftLayoutMetadata struct {
	Symlink  string `yaml:",omitempty"`
	Bind     string `yaml:",omitempty"`
	BindFile string `yaml:"bind_file,omitempty"`
	Type     string `yaml:",omitempty"`
}

// Snapcraft config.
type Snapcraft struct {
	NameTemplate string            `yaml:"name_template,omitempty"`
	Replacements map[string]string `yaml:",omitempty"`
	Publish      bool              `yaml:",omitempty"`

	ID          string                             `yaml:",omitempty"`
	Builds      []string                           `yaml:",omitempty"`
	Name        string                             `yaml:",omitempty"`
	Summary     string                             `yaml:",omitempty"`
	Description string                             `yaml:",omitempty"`
	Base        string                             `yaml:",omitempty"`
	License     string                             `yaml:",omitempty"`
	Grade       string                             `yaml:",omitempty"`
	Confinement string                             `yaml:",omitempty"`
	Layout      map[string]SnapcraftLayoutMetadata `yaml:",omitempty"`
	Apps        map[string]SnapcraftAppMetadata    `yaml:",omitempty"`
	Plugs       map[string]interface{}             `yaml:",omitempty"`

	Files []SnapcraftExtraFiles `yaml:"extra_files,omitempty"`
}

// SnapcraftExtraFiles config.
type SnapcraftExtraFiles struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination,omitempty"`
	Mode        uint32 `yaml:"mode,omitempty"`
}

// Snapshot config.
type Snapshot struct {
	NameTemplate string `yaml:"name_template,omitempty"`
}

// Checksum config.
type Checksum struct {
	NameTemplate string   `yaml:"name_template,omitempty"`
	Algorithm    string   `yaml:"algorithm,omitempty"`
	IDs          []string `yaml:"ids,omitempty"`
	Disable      bool     `yaml:"disable,omitempty"`
}

// Docker image config.
type Docker struct {
	Binaries           []string `yaml:",omitempty"` // deprecated: no need to use this anymore
	Builds             []string `yaml:",omitempty"` // deprecated: use IDs instead
	IDs                []string `yaml:"ids,omitempty"`
	Goos               string   `yaml:",omitempty"`
	Goarch             string   `yaml:",omitempty"`
	Goarm              string   `yaml:",omitempty"`
	Dockerfile         string   `yaml:",omitempty"`
	ImageTemplates     []string `yaml:"image_templates,omitempty"`
	SkipPush           string   `yaml:"skip_push,omitempty"`
	Files              []string `yaml:"extra_files,omitempty"`
	BuildFlagTemplates []string `yaml:"build_flag_templates,omitempty"`
	PushFlags          []string `yaml:"push_flags,omitempty"`
	Buildx             bool     `yaml:"use_buildx,omitempty"` // deprecated: use Use instead
	Use                string   `yaml:"use,omitempty"`
}

// DockerManifest config.
type DockerManifest struct {
	NameTemplate   string   `yaml:"name_template,omitempty"`
	SkipPush       string   `yaml:"skip_push,omitempty"`
	ImageTemplates []string `yaml:"image_templates,omitempty"`
	CreateFlags    []string `yaml:"create_flags,omitempty"`
	PushFlags      []string `yaml:"push_flags,omitempty"`
	Use            string   `yaml:"use,omitempty"`
}

// Filters config.
type Filters struct {
	Exclude []string `yaml:",omitempty"`
}

// Changelog Config.
type Changelog struct {
	Filters Filters `yaml:",omitempty"`
	Sort    string  `yaml:",omitempty"`
	Skip    bool    `yaml:",omitempty"`
}

// EnvFiles holds paths to files that contains environment variables
// values like the github token for example.
type EnvFiles struct {
	GitHubToken string `yaml:"github_token,omitempty"`
	GitLabToken string `yaml:"gitlab_token,omitempty"`
	GiteaToken  string `yaml:"gitea_token,omitempty"`
}

// Before config.
type Before struct {
	Hooks []string `yaml:",omitempty"`
}

// Blob contains config for GO CDK blob.
type Blob struct {
	Bucket     string      `yaml:",omitempty"`
	Provider   string      `yaml:",omitempty"`
	Region     string      `yaml:",omitempty"`
	DisableSSL bool        `yaml:"disableSSL,omitempty"`
	Folder     string      `yaml:",omitempty"`
	KMSKey     string      `yaml:",omitempty"`
	IDs        []string    `yaml:"ids,omitempty"`
	Endpoint   string      `yaml:",omitempty"` // used for minio for example
	ExtraFiles []ExtraFile `yaml:"extra_files,omitempty"`
}

// Upload configuration.
type Upload struct {
	Name               string            `yaml:",omitempty"`
	IDs                []string          `yaml:"ids,omitempty"`
	Target             string            `yaml:",omitempty"`
	Username           string            `yaml:",omitempty"`
	Mode               string            `yaml:",omitempty"`
	Method             string            `yaml:",omitempty"`
	ChecksumHeader     string            `yaml:"checksum_header,omitempty"`
	TrustedCerts       string            `yaml:"trusted_certificates,omitempty"`
	Checksum           bool              `yaml:",omitempty"`
	Signature          bool              `yaml:",omitempty"`
	CustomArtifactName bool              `yaml:"custom_artifact_name,omitempty"`
	CustomHeaders      map[string]string `yaml:"custom_headers,omitempty"`
}

// Publisher configuration.
type Publisher struct {
	Name      string   `yaml:",omitempty"`
	IDs       []string `yaml:"ids,omitempty"`
	Checksum  bool     `yaml:",omitempty"`
	Signature bool     `yaml:",omitempty"`
	Dir       string   `yaml:",omitempty"`
	Cmd       string   `yaml:",omitempty"`
	Env       []string `yaml:",omitempty"`
}

// Source configuration.
type Source struct {
	NameTemplate string `yaml:"name_template,omitempty"`
	Format       string `yaml:",omitempty"`
	Enabled      bool   `yaml:",omitempty"`
}

// Project includes all project configuration.
type Project struct {
	ProjectName     string           `yaml:"project_name,omitempty"`
	Env             []string         `yaml:",omitempty"`
	Release         Release          `yaml:",omitempty"`
	Milestones      []Milestone      `yaml:",omitempty"`
	Brews           []Homebrew       `yaml:",omitempty"`
	Scoop           Scoop            `yaml:",omitempty"`
	Builds          []Build          `yaml:",omitempty"`
	Archives        []Archive        `yaml:",omitempty"`
	NFPMs           []NFPM           `yaml:"nfpms,omitempty"`
	Snapcrafts      []Snapcraft      `yaml:",omitempty"`
	Snapshot        Snapshot         `yaml:",omitempty"`
	Checksum        Checksum         `yaml:",omitempty"`
	Dockers         []Docker         `yaml:",omitempty"`
	DockerManifests []DockerManifest `yaml:"docker_manifests,omitempty"`
	Artifactories   []Upload         `yaml:",omitempty"`
	Uploads         []Upload         `yaml:",omitempty"`
	Blobs           []Blob           `yaml:"blobs,omitempty"`
	Publishers      []Publisher      `yaml:"publishers,omitempty"`
	Changelog       Changelog        `yaml:",omitempty"`
	Dist            string           `yaml:",omitempty"`
	Signs           []Sign           `yaml:",omitempty"`
	EnvFiles        EnvFiles         `yaml:"env_files,omitempty"`
	Before          Before           `yaml:",omitempty"`
	Source          Source           `yaml:",omitempty"`
	GoMod           GoMod            `yaml:"gomod,omitempty"`
	Announce        Announce         `yaml:"announce,omitempty"`

	// this is a hack ¯\_(ツ)_/¯
	SingleBuild Build `yaml:"build,omitempty"`

	// should be set if using github enterprise
	GitHubURLs GitHubURLs `yaml:"github_urls,omitempty"`

	// should be set if using a private gitlab
	GitLabURLs GitLabURLs `yaml:"gitlab_urls,omitempty"`

	// should be set if using Gitea
	GiteaURLs GiteaURLs `yaml:"gitea_urls,omitempty"`
}

type GoMod struct {
	Proxy    bool     `yaml:",omitempty"`
	Env      []string `yaml:",omitempty"`
	GoBinary string   `yaml:",omitempty"`
}

type Announce struct {
	Twitter Twitter `yaml:"twitter,omitempty"`
}

type Twitter struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	MessageTemplate string `yaml:"message_template,omitempty"`
}

// Load config file.
func Load(file string) (config Project, err error) {
	f, err := os.Open(file) // #nosec
	if err != nil {
		return
	}
	defer f.Close()
	log.WithField("file", file).Info("loading config file")
	return LoadReader(f)
}

// LoadReader config via io.Reader.
func LoadReader(fd io.Reader) (config Project, err error) {
	data, err := io.ReadAll(fd)
	if err != nil {
		return config, err
	}
	err = yaml.UnmarshalStrict(data, &config)
	log.WithField("config", config).Debug("loaded config file")
	return config, err
}
