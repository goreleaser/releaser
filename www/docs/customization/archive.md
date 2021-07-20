---
title: Archive
---

The binaries built will be archived together with the `README` and `LICENSE` files into a
`tar.gz` file. In the `archives` section you can customize the archive name,
additional files, and format.

Here is a commented `archives` section with all fields specified:

```yaml
# .goreleaser.yml
archives:
  -
    # ID of this archive.
    # Defaults to `default`.
    id: my-archive

    # Builds reference which build instances should be archived in this archive.
    builds:
    - default

    # Archive format. Valid options are `tar.gz`, `tar.xz`, `gz`, `zip` and `binary`.
    # If format is `binary`, no archives are created and the binaries are instead
    # uploaded directly.
    # Default is `tar.gz`.
    format: zip

    # Archive name template.
    # Defaults:
    # - if format is `tar.gz`, `tar.xz`, `gz` or `zip`:
    #   - `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}`
    # - if format is `binary`:
    #   - `{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}`
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # Replacements for GOOS and GOARCH in the archive name.
    # Keys should be valid GOOSs or GOARCHs.
    # Values are the respective replacements.
    # Default is empty.
    replacements:
      amd64: 64-bit
      386: 32-bit
      darwin: macOS
      linux: Tux

    # Set to true, if you want all files in the archive to be in a single directory.
    # If set to true and you extract the archive 'goreleaser_Linux_arm64.tar.gz',
    # you get a folder 'goreleaser_Linux_arm64'.
    # If set to false, all files are extracted separately.
    # You can also set it to a custom folder name (templating is supported).
    # Default is false.
    wrap_in_directory: true

    # Can be used to change the archive formats for specific GOOSs.
    # Most common use case is to archive as zip on Windows.
    # Default is empty.
    format_overrides:
      - goos: windows
        format: zip

    # Additional files/template/globs you want to add to the archive.
    # Defaults are any files matching `LICENSE*`, `README*`, `CHANGELOG*`,
    #  `license*`, `readme*` and `changelog*`.
    files:
      - LICENSE.txt
      - README_{{.Os}}.md
      - CHANGELOG.md
      - docs/*
      - design/*.png
      - templates/**/*
      # a more complete example, check the globbing deep dive bellow
      - src: '*.md'
        dst: docs
        # Strip parent folders when adding files to the archive.
        # Default: false
        strip_parent: true
        # File info.
        # Not all fields are supported by all formats available formats.
        # Defaults to the file info of the actual file if not provided.
        info:
          owner: root
          group: root
          mode: 0644
          # format is `time.RFC3339Nano`
          mtime: 2008-01-02T15:04:05Z

    # Disables the binary count check.
    # Default: false
    allow_different_binary_count: true
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

!!! tip
    You can add entire folders, its subfolders and files by using the glob notation,
    for example: `myfolder/**/*`.

!!! warning
    The `files` and `wrap_in_directory` options are ignored if `format` is `binary`.

!!! warning
    The `name_template` option will not reflect the filenames under the `dist` folder if `format` is `binary`.
    The template will be applied only where the binaries are uploaded (e.g. GitHub releases).

## Deep diving into the globbing options

We'll walk through what happens in each case using some examples.

```yaml
# ...
files:

# Adds `README.md` at the root of the archive:
- README.md

# Adds all `md` files to the root of the archive:
- '*.md'

# Adds all `md` files to the root of the archive:
- src: '*.md'

# Adds all `md` files in the current folder to a `docs` folder in the archive:
- src: '*.md'
  dst: docs

# Recursively adds all `go` files to a `source` folder in the archive.
# in this case, `cmd/myapp/main.go` will be added as `source/cmd/myapp/main.go`
- src: '**/*.go'
  dst: source

# Recursively adds all `go` files to a `source` folder in the archive, stripping their parent folder.
# In this case, `cmd/myapp/main.go` will be added as `source/main.go`:
- src: '**/*.go'
  dst: source
  strip_parent: true
# ...
```

!!! warning
    `strip_parent` is only effective if `dst` is not empty.

## Packaging only the binaries

Since GoReleaser will always add the `README` and `LICENSE` files to the
archive if the file list is empty, you'll need to provide a filled `files`
on the archive section.

A working hack is to use something like this:

```yaml
# .goreleaser.yml
archives:
- files:
  - none*
```

This would add all files matching the glob `none*`, provide that you don't
have any files matching that glob, only the binary will be added to the
archive.

For more information, check [#602](https://github.com/goreleaser/goreleaser/issues/602)

## A note about Gzip

Gzip is a compression-only format, therefore, it couldn't have more than one
file inside.

Presumably, you'll want that file to be the binary, so, your archive section
will probably look like this:

```yaml
# .goreleaser.yml
archives:
- format: gz
  files:
  - none*
```

This should create `.gz` files with the binaries only, which should be
extracted with something like `gzip -d file.gz`.

!!! warning
    You won't be able to package multiple builds in a single archive either.
    The alternative is to declare multiple archives filtering by build ID.

## Disable archiving

You can do that by setting `format` to `binary`:

```yaml
# .goreleaser.yml
archives:
- format: binary
```

Make sure to check the rest of the documentation above, as doing this has some
implications.
