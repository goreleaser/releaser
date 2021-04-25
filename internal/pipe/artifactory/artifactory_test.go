package artifactory

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var (
	// mux is the HTTP request multiplexer used with the test server.
	mux *http.ServeMux

	// server is a test HTTP server used to provide mock API responses.
	server *httptest.Server
)

func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
}

// teardown closes the test HTTP server.
func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header, want string) {
	t.Helper()
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %q, want %q", header, got, want)
	}
}

// TODO: improve all tests bellow by checking wether the mocked handlers
// were called or not.

func TestRunPipe_ModeBinary(t *testing.T) {
	setup()
	defer teardown()

	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0o666))

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "/mybin/darwin/amd64/mybin",
			"created" : "2017-12-02T19:30:45.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/mybin/darwin/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/example-repo-local/mybin/darwin/amd64/mybin"
		  }`)
	})
	mux.HandleFunc("/example-repo-local/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "mybin/linux/amd64/mybin",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/mybin/linux/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/example-repo-local/mybin/linux/amd64/mybin"
		  }`)
	})
	mux.HandleFunc("/production-repo-remote/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "production-repo-remote",
			"path" : "mybin/darwin/amd64/mybin",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "productionuser",
			"downloadUri" : "http://127.0.0.1:56563/production-repo-remote/mybin/darwin/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/production-repo-remote/mybin/darwin/amd64/mybin"
		  }`)
	})
	mux.HandleFunc("/production-repo-remote/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "production-repo-remote",
			"path" : "mybin/linux/amd64/mybin",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "productionuser",
			"downloadUri" : "http://127.0.0.1:56563/production-repo-remote/mybin/linux/amd64/mybin",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/production-repo-remote/mybin/linux/amd64/mybin"
		  }`)
	})

	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Upload{
			{
				Name:     "production-us",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
			{
				Name:     "production-eu",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/production-repo-remote/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "productionuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION-US_SECRET": "deployuser-secret",
		"ARTIFACTORY_PRODUCTION-EU_SECRET": "productionuser-apikey",
	}
	for _, goos := range []string{"linux", "darwin"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Path:   binPath,
			Goarch: "amd64",
			Goos:   goos,
			Type:   artifact.UploadableBinary,
		})
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestRunPipe_ModeArchive(t *testing.T) {
	setup()
	defer teardown()

	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())

	ctx := context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Version }}/", server.URL),
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Version = "1.0.0"
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.LinuxPackage,
		Name: "bin.deb",
		Path: debfile.Name(),
	})

	var uploads sync.Map

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "/goreleaser/bin.tar.gz",
			"created" : "2017-12-02T19:30:45.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.tar.gz",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.tar.gz"
		  }`)
		uploads.Store("targz", true)
	})
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.deb", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"repo" : "example-repo-local",
			"path" : "goreleaser/bin.deb",
			"created" : "2017-12-02T19:30:46.436Z",
			"createdBy" : "deployuser",
			"downloadUri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.deb",
			"mimeType" : "application/octet-stream",
			"size" : "9",
			"checksums" : {
			  "sha1" : "65d01857a69f14ade727fe1ceee0f52a264b6e57",
			  "md5" : "a55e303e7327dc871a8e2a84f30b9983",
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"originalChecksums" : {
			  "sha256" : "ead9b172aec5c24ca6c12e85a1e6fc48dd341d8fac38c5ba00a78881eabccf0e"
			},
			"uri" : "http://127.0.0.1:56563/example-repo-local/goreleaser/bin.deb"
		  }`)
		uploads.Store("deb", true)
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
	_, ok := uploads.Load("targz")
	require.True(t, ok, "tar.gz file was not uploaded")
	_, ok = uploads.Load("deb")
	require.True(t, ok, "deb file was not uploaded")
}

func TestRunPipe_ArtifactoryDown(t *testing.T) {
	folder := t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())

	ctx := context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "archive",
				Target:   "http://localhost:1234/example-repo-local/{{ .ProjectName }}/{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})
	ctx.Version = "2.0.0"
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err = Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection refused")
}

func TestRunPipe_TargetTemplateError(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	binPath := filepath.Join(dist, "mybin", "mybin")

	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Upload{
			{
				Name: "production",
				Mode: "binary",
				// This template is not correct and should fail
				Target:   "http://storage.company.com/example-repo-local/{{ .ProjectName /{{ .Version }}/",
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Publish(ctx), `artifactory: error while building the target url: template: tmpl:1: unexpected "/" in operand`)
}

func TestRunPipe_BadCredentials(t *testing.T) {
	setup()
	defer teardown()

	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0o666))

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{
			"errors" : [ {
			  "status" : 401,
			  "message" : "Bad credentials"
			} ]
		  }`)
	})

	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Bad credentials")
}

func TestRunPipe_UnparsableErrorResponse(t *testing.T) {
	setup()
	defer teardown()

	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0o666))

	// Dummy artifactories
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `...{
			"errors" : [ {
			 ...
			} ]
		  }`)
	})

	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Publish(ctx), `artifactory: upload failed: invalid character '.' looking for beginning of value`)
}

func TestRunPipe_FileNotFound(t *testing.T) {
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        "archivetest/dist",
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "archivetest/dist/mybin/mybin",
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err:=Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestRunPipe_UnparsableTarget(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0o666))

	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Publish(ctx), `artifactory: upload failed: parse "://artifacts.company.com/example-repo-local/mybin/darwin/amd64/mybin": missing protocol scheme`)
}

func TestRunPipe_SkipWhenPublishFalse(t *testing.T) {
	ctx := context.New(config.Project{
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.SkipPublish = true

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Publish(ctx)
	require.True(t, pipe.IsSkip(err))
	require.EqualError(t, err, pipe.ErrSkipPublishEnabled.Error())
}

func TestRunPipe_DirUpload(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin")

	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Mode:     "binary",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Dir(binPath),
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Publish(ctx), `artifactory: upload failed: the asset to upload can't be a directory`)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNoArtifactories(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestArtifactoriesWithoutTarget(t *testing.T) {
	ctx := &context.Context{
		Env: map[string]string{
			"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Artifactories: []config.Upload{
				{
					Name:     "production",
					Username: "deployuser",
				},
			},
		},
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestArtifactoriesWithoutUsername(t *testing.T) {
	ctx := &context.Context{
		Env: map[string]string{
			"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Artifactories: []config.Upload{
				{
					Name:   "production",
					Target: "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				},
			},
		},
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestArtifactoriesWithoutName(t *testing.T) {
	ctx := context.New(config.Project{
		Artifactories: []config.Upload{
			{
				Username: "deployuser",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestArtifactoriesWithoutSecret(t *testing.T) {
	ctx := context.New(config.Project{
		Artifactories: []config.Upload{
			{
				Name:     "production",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestArtifactoriesWithInvalidMode(t *testing.T) {
	ctx := &context.Context{
		Env: map[string]string{
			"ARTIFACTORY_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Artifactories: []config.Upload{
				{
					Name:     "production",
					Mode:     "does-not-exists",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.Error(t, Pipe{}.Publish(ctx))
}

func TestDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Artifactories: []config.Upload{
				{
					Name:     "production",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Artifactories, 1)
	artifactory := ctx.Config.Artifactories[0]
	require.Equal(t, "archive", artifactory.Mode)
}

func TestDefaultNoArtifactories(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Artifactories: []config.Upload{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Artifactories)
}

func TestDefaultSet(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Artifactories: []config.Upload{
				{
					Mode:           "custom",
					ChecksumHeader: "foo",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Artifactories, 1)
	artifactory := ctx.Config.Artifactories[0]
	require.Equal(t, "custom", artifactory.Mode)
	require.Equal(t, "foo", artifactory.ChecksumHeader)
}
