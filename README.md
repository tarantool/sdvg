# Synthetic Data Values Generator (SDVG)

## Language

- **English**
- [Русский](README.ru.md)

## Description

SDVG (Synthetic Data Values Generator) is a tool for generating synthetic data.
It supports various run modes, data types for generation, and output formats.

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
./sdvg generate simple_model.yml
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
./sdvg
```

To view available commands and arguments:

```bash
./sdvg -h
./sdvg --help
./sdvg generate -h
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
