# Developer Guide

## Development

To support a new data destination, you need to implement a module in
the [writer](../../internal/generator/output/general/writer) directory of `output` layer.

## Testing

For static analysis, we use the `golangci-lint` linter.
Run it with `make test/lint`.
You can also automatically fix some linting issues with `make test/lint/fix`.

For unit testing, standard Go tests are used.
Run them with `make test/unit`.

For performance testing, standard Go tests are used as well.
Run them with `make test/performance`.

To calculate code coverage, run `make test/cover`.

## Build

To build the generator for your current OS, run `make build`.
The resulting binaries will be in the `build/out` directory.

## Release

To release a new version:

1. Review and update the changelog in [CHANGELOG.md](../../CHANGELOG.md) if needed.
2. Add a new header for the release version with a link to the git diff in [CHANGELOG.md](../../CHANGELOG.md).
   Use the format: `## [0.0.0](https://.../compare/prev...current) - 2000-12-31`;
3. Commit to the main branch (via MR) with a commit message containing `release 0.0.0`.
