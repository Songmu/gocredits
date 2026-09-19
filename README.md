gocredits
=======

[![Test Status](https://github.com/Songmu/gocredits/workflows/test/badge.svg?branch=main)][actions]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Songmu/gocredits)][PkgGoDev]

[actions]: https://github.com/Songmu/gocredits/actions?workflow=test
[license]: https://github.com/Songmu/gocredits/blob/main/LICENSE
[PkgGoDev]: https://pkg.go.dev/github.com/Songmu/gocredits

gocredits creates CREDITS file from LICENSE files of dependencies

## Synopsis

```console
gocredits . > CREDITS
```

## Description

When distributing built executable in Go, we need to include LICENSE of the dependent
libraries into the package, so gocredits bundle them together as a CREDITS file.

To use `gocredits`, we should use go modules for dependency management.

## Installation

### homebrew

```console
% brew install Songmu/tap/gocredits
```

### go get

```console
% go install github.com/Songmu/gocredits/cmd/gocredits@latest
```

### [aqua](https://aquaproj.github.io/)

```console
% aqua g -i Songmu/gocredits
```

Built binaries are available on GitHub Releases.
<https://github.com/Songmu/gocredits/releases>

## GitHub Actions

The action installs and runs gocredits, overwriting `CREDITS` in the repository
root:

```yaml
permissions:
  attestations: read
  contents: read

steps:
- uses: actions/checkout@v7
- id: gocredits
  uses: Songmu/gocredits@v0
```

The module directory, missing-license behavior, and output format can be
configured:

```yaml
- uses: Songmu/gocredits@v0
  with:
    directory: ./path/to/module
    skip-missing: true
    format: |
      {{range .Licenses}}{{.Name}}
      {{end}}
```

The action outputs the absolute `credits` path and a `changed` value indicating
whether the file contents changed. It does not commit or push the generated
file. The `attestations: read` permission allows the action to verify the
downloaded gocredits binary. The `contents: read` permission is required by the
checkout step.

## Author

[Songmu](https://github.com/Songmu)
