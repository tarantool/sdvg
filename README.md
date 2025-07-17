# Synthetic Data Values Generator (SDVG)

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

## Documentation

- [Русская документация](./doc/ru/index.md)
- [English documentation](./doc/en/index.md)
- [Changelog](./CHANGELOG.md)
- [License](./LICENSE)

## Maintainers

- [@hackallcode](https://github.com/hackallcode)
- [@ReverseTM](https://github.com/ReverseTM)
- [@choseenonee](https://github.com/choseenonee)
- [@Hoodie-Huuuuu](https://github.com/Hoodie-Huuuuu)
