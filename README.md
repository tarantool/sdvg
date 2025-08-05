<div class="hide-in-mkdocs">

# Synthetic Data Values Generator (SDVG)

</div>

[![Release][release-badge]][release-url]
[![Pre-release][pre-release-badge]][pre-release-url]
[![CI][actions-badge]][actions-url]
[![Coverage Status][test-coverage-badge]][test-coverage-url]
[![Language][language-badge]][language-url]
[![License][license-badge]][license-url]

[release-badge]: https://img.shields.io/github/v/release/tarantool/sdvg
[release-url]: https://github.com/tarantool/sdvg/releases/latest/
[pre-release-badge]: https://img.shields.io/badge/pre--release-latest-orange
[pre-release-url]: https://github.com/tarantool/sdvg/releases/tag/latest/
[actions-badge]: https://img.shields.io/github/check-runs/tarantool/sdvg/master
[actions-url]: https://github.com/tarantool/sdvg/actions
[test-coverage-badge]: https://img.shields.io/coverallsCoverage/github/tarantool/sdvg?branch=master
[test-coverage-url]: https://coveralls.io/github/tarantool/sdvg?branch=master
[language-badge]: https://img.shields.io/github/languages/top/tarantool/sdvg
[language-url]: https://github.com/tarantool/sdvg/search?l=go
[license-badge]: https://img.shields.io/github/license/tarantool/sdvg
[license-url]: ./LICENSE

<div class="hide-in-mkdocs">

## Documentation version
- [Multilingual web version](https://tarantool.github.io/sdvg/) (recommended)
- **English**
- [Русский](README.ru.md)

</div>

## Description

SDVG (Synthetic Data Values Generator) is a tool for generating synthetic data.
It supports various run modes, data types for generation, and output formats.

![scheme.png](./asset/scheme.png)

Run modes:

- CLI - generate data, create configs, and validate them via the console;
- HTTP server - accepts generation requests through an HTTP API.

Data types:

- strings (english, russian);
- integers and floating-point numbers;
- dates with timestamps;
- UUID.

String subtypes:

- random strings;
- texts;
- first names;
- last names;
- phone numbers;
- patterns.

Each data type can be generated with the following options:

- specify percentage/number of unique values per column;
- ordered generation (sequence);
- foreign key reference;
- idempotent generation using a seed number;
- value generation from ranges with percentage-based distribution.

Output formats:

- devnull;
- CSV files;
- Parquet files;
- HTTP API;
- Tarantool Column Store HTTP API.

## Installation

### Standard installation

You can install SDVG by downloading the appropriate binary version
from the [GitHub Releases page](https://github.com/tarantool/sdvg/releases).

Download binary for your OS:

```shell
# Linux (x86-64)
curl -Lo sdvg https://github.com/tarantool/sdvg/releases/latest/download/sdvg-linux-amd64
```

```shell
# Linux (ARM64)
curl -Lo sdvg https://github.com/tarantool/sdvg/releases/latest/download/sdvg-linux-arm64
```

```shell
# macOS (x86-64)
curl -Lo sdvg https://github.com/tarantool/sdvg/releases/latest/download/sdvg-darwin-amd64
```

```shell
# macOS (ARM64)
curl -Lo sdvg https://github.com/tarantool/sdvg/releases/latest/download/sdvg-darwin-arm64
```

Install binary in your system:

```shell
chmod +x sdvg
sudo mv sdvg /usr/local/bin/sdvg
```

Check that everything works correctly:

```shell
sdvg version
```

### Compile and install from sources

To compile and install this tool, you can use `go install` command:

```shell
# To get the specified version
go install github.com/tarantool/sdvg@0.0.2
# To get a version from the master branch
go clean -modcache
go install github.com/tarantool/sdvg@latest
```

Check that everything works correctly:

```shell
sdvg version
```

## Quick Start

Here's an example of a data model that generates 10,000 user rows and writes them to a CSV file:

```yaml
output:
  type: csv
models:
  user:
    rows_count: 10000
    columns:
      - name: id
        type: uuid
      - name: name
        type: string
        type_params:
          logical_type: first_name
```

Save this as `simple_model.yml`, then run:

```bash
sdvg generate simple_model.yml
```

This will create a CSV file with fake user data like `id` and `name`:

```csv
id,name
c8a53cfd-1089-4154-9627-560fbbea2fef,Sutherlan
b5c024f8-3f6f-43d3-b021-0bb2305cc680,Hilton
5adf8218-7b53-41bb-873d-c5768ca6afa2,Craggy
...
```

To launch the generator in interactive mode:

```bash
sdvg
```

To view available commands and arguments:

```bash
sdvg -h
sdvg --help
sdvg generate -h
```

More information can be found in the [user guide](./doc/en/usage.md).

## Documentation

- [User Guide](./doc/en/usage.md)
- [Developer Guide](./doc/en/contributing.md)
- [Goals and Standards Compliance](./doc/en/overview.md)
- [Changelog](./CHANGELOG.md)
- [License](./LICENSE)

## Maintainers

- [@hackallcode](https://github.com/hackallcode)
- [@ReverseTM](https://github.com/ReverseTM)
- [@choseenonee](https://github.com/choseenonee)
- [@Hoodie-Huuuuu](https://github.com/Hoodie-Huuuuu)
