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

## Documentation

To edit the documentation, you need to make changes for each supported language
within its corresponding directory in the [doc](../../doc) directory.

To add a new section to the documentation, you must:

1. Create a new `.md` file in the root of the [doc/en](../../doc/en) directory.
2. Add the file name to the `nav` section in the [mkdocs.yml](../../mkdocs.yml) configuration file.
3. Perform the same steps for other languages if they need to be supported.
   Otherwise, the English version of the section will be displayed.
4. Translate the section titles for other languages in the `plugins.i18n.languages.<language_code>.nav_translations`
   section of [mkdocs.yml](../../mkdocs.yml).

To locally check the documentation site's layout, you need to:

1. Install the Python dependencies: `make doc/prepare`.
2. Run the local site hosting with the command `make doc/serve`.
   The site will be available at [127.0.0.1:8000](http://127.0.0.1:8000).

## Release

To release a new version:

1. Review and update the changelog in [CHANGELOG.md](../../CHANGELOG.md) if needed.
2. Add a new header for the release version with a link to the git diff in [CHANGELOG.md](../../CHANGELOG.md).
   Use the format: `## [0.0.0](https://.../compare/prev...current) - 2000-12-31`.
3. Commit to the main branch (via MR) with a commit message containing `release 0.0.0`.
