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

Built binaries are available on GitHub Releases.
<https://github.com/Songmu/gocredits/releases>

## Author

[Songmu](https://github.com/Songmu)
