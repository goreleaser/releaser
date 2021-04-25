package upload

import (
	"fmt"
	"net/http"
	h "net/http"
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

	var folder = t.TempDir()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0666))

	// Dummy http server
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/production-repo-remote/mybin/linux/amd64/mybin")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/example-repo-local/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/production-repo-remote/mybin/linux/amd64/mybin")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/production-repo-remote/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.Header().Set("Location", "/production-repo-remote/mybin/linux/amd64/mybin")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/production-repo-remote/mybin/linux/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "productionuser" with secret "productionuser-apikey"
		testHeader(t, r, "Authorization", "Basic cHJvZHVjdGlvbnVzZXI6cHJvZHVjdGlvbnVzZXItYXBpa2V5")

		w.Header().Set("Location", "/production-repo-remote/mybin/linux/amd64/mybin")
		w.WriteHeader(http.StatusCreated)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
				Name:     "production-us",
				Mode:     "binary",
				Target:   fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}", server.URL),
				Username: "deployuser",
			},
			{
				Method:   h.MethodPut,
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
		"UPLOAD_PRODUCTION-US_SECRET": "deployuser-secret",
		"UPLOAD_PRODUCTION-EU_SECRET": "productionuser-apikey",
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

	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestRunPipe_ModeArchive(t *testing.T) {
	setup()
	defer teardown()

	var folder = t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())

	var ctx = context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
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

	// Dummy http server
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/example-repo-local/goreleaser/1.0.0/bin.tar.gz")
		w.WriteHeader(http.StatusCreated)
		uploads.Store("targz", true)
	})
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.deb", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/example-repo-local/goreleaser/1.0.0/bin.deb")
		w.WriteHeader(http.StatusCreated)
		uploads.Store("deb", true)
	})

	require.NoError(t, Pipe{}.Publish(ctx))
	_, ok := uploads.Load("targz")
	require.True(t, ok, "tar.gz file was not uploaded")
	_, ok = uploads.Load("deb")
	require.True(t, ok, "deb file was not uploaded")
}

func TestRunPipe_ModeBinary_CustomArtifactName(t *testing.T) {
	setup()
	defer teardown()

	var folder = t.TempDir()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0666))

	// Dummy http server
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin;deb.distribution=xenial", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/production-repo-remote/mybin/linux/amd64/mybin;deb.distribution=xenial")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/example-repo-local/mybin/linux/amd64/mybin;deb.distribution=xenial", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/example-repo-local/mybin/linux/amd64/mybin;deb.distribution=xenial")
		w.WriteHeader(http.StatusCreated)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Uploads: []config.Upload{
			{
				Method:             h.MethodPut,
				Name:               "production-us",
				Mode:               "binary",
				Target:             fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}/{{ .ArtifactName }};deb.distribution=xenial", server.URL),
				Username:           "deployuser",
				CustomArtifactName: true,
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"UPLOAD_PRODUCTION-US_SECRET": "deployuser-secret",
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

	require.NoError(t, Pipe{}.Publish(ctx))
}

func TestRunPipe_ModeArchive_CustomArtifactName(t *testing.T) {
	setup()
	defer teardown()

	var folder = t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())
	debfile, err := os.Create(filepath.Join(folder, "bin.deb"))
	require.NoError(t, err)
	require.NoError(t, debfile.Close())

	var ctx = context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		Uploads: []config.Upload{
			{
				Method:             h.MethodPut,
				Name:               "production",
				Mode:               "archive",
				Target:             fmt.Sprintf("%s/example-repo-local/{{ .ProjectName }}/{{ .Version }}/{{ .ArtifactName }};deb.distribution=xenial", server.URL),
				Username:           "deployuser",
				CustomArtifactName: true,
			},
		},
		Archives: []config.Archive{
			{},
		},
	})
	ctx.Env = map[string]string{
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
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

	// Dummy http server
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.tar.gz;deb.distribution=xenial", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/example-repo-local/goreleaser/1.0.0/bin.tar.gz;deb.distribution=xenial")
		w.WriteHeader(http.StatusCreated)
		uploads.Store("targz", true)
	})
	mux.HandleFunc("/example-repo-local/goreleaser/1.0.0/bin.deb;deb.distribution=xenial", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.Header().Set("Location", "/example-repo-local/goreleaser/1.0.0/bin.deb;deb.distribution=xenial")
		w.WriteHeader(http.StatusCreated)
		uploads.Store("deb", true)
	})

	require.NoError(t, Pipe{}.Publish(ctx))
	_, ok := uploads.Load("targz")
	require.True(t, ok, "tar.gz file was not uploaded")
	_, ok = uploads.Load("deb")
	require.True(t, ok, "deb file was not uploaded")
}

func TestRunPipe_ArtifactoryDown(t *testing.T) {
	var folder = t.TempDir()
	tarfile, err := os.Create(filepath.Join(folder, "bin.tar.gz"))
	require.NoError(t, err)
	require.NoError(t, tarfile.Close())

	var ctx = context.New(config.Project{
		ProjectName: "goreleaser",
		Dist:        folder,
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
				Name:     "production",
				Mode:     "archive",
				Target:   "http://localhost:1234/example-repo-local/{{ .ProjectName }}/{{ .Version }}/",
				Username: "deployuser",
			},
		},
	})
	ctx.Version = "2.0.0"
	ctx.Env = map[string]string{
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.UploadableArchive,
		Name: "bin.tar.gz",
		Path: tarfile.Name(),
	})
	err = Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection refused")
}

func TestRunPipe_TargetTemplateError(t *testing.T) {
	var folder = t.TempDir()
	var dist = filepath.Join(folder, "dist")
	var binPath = filepath.Join(dist, "mybin", "mybin")

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Uploads: []config.Upload{
			{
				Method: h.MethodPut,
				Name:   "production",
				Mode:   "binary",
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})
	var err = Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), `upload: error while building the target url: template: tmpl:1: unexpected "/" in operand`)
}

func TestRunPipe_BadCredentials(t *testing.T) {
	setup()
	defer teardown()

	var folder = t.TempDir()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0666))

	// Dummy http server
	mux.HandleFunc("/example-repo-local/mybin/darwin/amd64/mybin", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		testHeader(t, r, "Content-Length", "9")
		// Basic auth of user "deployuser" with secret "deployuser-secret"
		testHeader(t, r, "Authorization", "Basic ZGVwbG95dXNlcjpkZXBsb3l1c2VyLXNlY3JldA==")

		w.WriteHeader(http.StatusUnauthorized)
	})

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	var err = Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unauthorized")
}

func TestRunPipe_FileNotFound(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        "archivetest/dist",
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "archivetest/dist/mybin/mybin",
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	err:=Pipe{}.Publish(ctx)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestRunPipe_UnparsableTarget(t *testing.T) {
	var folder = t.TempDir()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	d1 := []byte("hello\ngo\n")
	require.NoError(t, os.WriteFile(binPath, d1, 0666))

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.EqualError(t, Pipe{}.Publish(ctx), `upload: upload failed: parse "://artifacts.company.com/example-repo-local/mybin/darwin/amd64/mybin": missing protocol scheme`)
}

func TestRunPipe_SkipWhenPublishFalse(t *testing.T) {
	var ctx = context.New(config.Project{
		Uploads: []config.Upload{
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.SkipPublish = true

	err := Pipe{}.Publish(ctx)
	require.True(t, pipe.IsSkip(err))
	require.EqualError(t, err, pipe.ErrSkipPublishEnabled.Error())
}

func TestRunPipe_DirUpload(t *testing.T) {
	var folder = t.TempDir()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin")

	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
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
		"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Dir(binPath),
		Goarch: "amd64",
		Goos:   "darwin",
		Type:   artifact.UploadableBinary,
	})

	require.EqualError(t, Pipe{}.Publish(ctx), `upload: upload failed: the asset to upload can't be a directory`)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNoPuts(t *testing.T) {
	require.True(t, pipe.IsSkip(Pipe{}.Publish(context.New(config.Project{}))))
}

func TestPutsWithoutTarget(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Uploads: []config.Upload{
				{
					Method:   h.MethodPut,
					Name:     "production",
					Username: "deployuser",
				},
			},
		},
	}

	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestPutsWithoutUsername(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Uploads: []config.Upload{
				{
					Method: h.MethodPut,
					Name:   "production",
					Target: "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				},
			},
		},
	}

	require.True(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestPutsWithoutName(t *testing.T) {
	require.True(t, pipe.IsSkip(Pipe{}.Publish(context.New(config.Project{
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
				Username: "deployuser",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
			},
		},
	}))))
}

func TestPutsWithoutSecret(t *testing.T) {
	require.True(t, pipe.IsSkip(Pipe{}.Publish(context.New(config.Project{
		Uploads: []config.Upload{
			{
				Method:   h.MethodPut,
				Name:     "production",
				Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
				Username: "deployuser",
			},
		},
	}))))
}

func TestPutsWithInvalidMode(t *testing.T) {
	var ctx = &context.Context{
		Env: map[string]string{
			"UPLOAD_PRODUCTION_SECRET": "deployuser-secret",
		},
		Config: config.Project{
			Uploads: []config.Upload{
				{
					Method:   h.MethodPut,
					Name:     "production",
					Mode:     "does-not-exists",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}
	require.Error(t, Pipe{}.Publish(ctx))
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Uploads: []config.Upload{
				{
					Name:     "production",
					Target:   "http://artifacts.company.com/example-repo-local/{{ .ProjectName }}/{{ .Os }}/{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}",
					Username: "deployuser",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Uploads, 1)
	var upload = ctx.Config.Uploads[0]
	require.Equal(t, "archive", upload.Mode)
	require.Equal(t, h.MethodPut, upload.Method)
}

func TestDefaultNoPuts(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Uploads: []config.Upload{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Uploads)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Uploads: []config.Upload{
				{
					Method: h.MethodPost,
					Mode:   "custom",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Uploads, 1)
	var upload = ctx.Config.Uploads[0]
	require.Equal(t, "custom", upload.Mode)
	require.Equal(t, h.MethodPost, upload.Method)
}
