gocredits
=======

[![Build Status](https://travis-ci.org/Songmu/gocredits.svg?branch=master)][travis]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]
[![GoDoc](https://godoc.org/github.com/Songmu/gocredits?status.svg)][godoc]

[travis]: https://travis-ci.org/Songmu/gocredits
[coveralls]: https://coveralls.io/r/Songmu/gocredits?branch=master
[license]: https://github.com/Songmu/gocredits/blob/master/LICENSE
[godoc]: https://godoc.org/github.com/Songmu/gocredits

gocredits creates CREDITS file from LICENSE files of dependencies

## Synopsis

```console
gocredits -w .
```

## Description

When distributing built executable in Go, we need to include LICENSE of the dependent
libraries into the package, so gocredits bundle them together as a CREDITS file.

To use `gocredits`, we should use go modules for dependency management.

## Installation

```console
% go get github.com/Songmu/gocredits
```

## Author

[Songmu](https://github.com/Songmu)
