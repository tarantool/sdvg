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
3. Perform the same steps in the corresponding directories for each language if you need to support them.
   Otherwise, the English version of the section will be displayed.

To add support for a new language, you need to:

1. Create a directory in the root of the [doc](../../doc) directory. The directory name must match
   the language code (e.g., ru, en, ...) that you want to support.
2. Translate the content of all the sections that you want to support in the new language.
   For everything to work correctly, the filenames of the translated documentation must match
   the names in the `doc/en` directory. All untranslated files will be replaced with the English version.
3. Add the new language to the `plugins.i18n.languages` section in the `mkdocs.yml` configuration file,
   where the `locale` property must match the name of the directory for the corresponding language.

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
