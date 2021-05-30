package release

import (
	"bytes"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const bodyTemplateText = `{{- with .Header }}{{ . }}

{{ end }}
{{- .ReleaseNotes }}

{{- with .DockerImages }}

## Docker images
{{ range $element := . }}
- ` + "`docker pull {{ . -}}`" + `
{{- end -}}
{{- end }}
{{- with .Footer }}{{ . }}{{ end }}
`

func describeBody(ctx *context.Context) (bytes.Buffer, error) {
	var out bytes.Buffer
	// nolint:prealloc
	var dockers []string
	for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerManifest)).List() {
		dockers = append(dockers, a.Name)
	}
	if len(dockers) == 0 {
		for _, a := range ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImage)).List() {
			dockers = append(dockers, a.Name)
		}
	}
	bodyTemplate := template.Must(template.New("release").Parse(bodyTemplateText))
	err := bodyTemplate.Execute(&out, struct {
		Header       string
		Footer       string
		ReleaseNotes string
		DockerImages []string
	}{
		Header:       ctx.Config.Release.Header,
		Footer:       ctx.Config.Release.Footer,
		ReleaseNotes: ctx.ReleaseNotes,
		DockerImages: dockers,
	})
	return out, err
}
