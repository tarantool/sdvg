# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [latest](https://github.com/tarantool/sdvg/compare/0.0.1..master)

## [0.0.1](https://github.com/tarantool/sdvg/compare/36d0930..0.0.1) - 2025-07-21

### Added

- CLI command to generate data
- CLI command to generate generation config
- CLI command to validate config
- CLI command to serve HTTP API for generator
- Progress in CLI (as logs or as progress bars)
- Flag to enable debug mode
- Flag for forced data generation with deletion of conflicting files
- Flag to enable CPU profiling
- String, integer, float, UUID, Datetime types
- Logical types for strings: first and last names, phones, texts
- Locales for logical types: ru, en
- String templates
- Unique values generation
- Nullable values generation
- Foreign keys generation
- Ordered values generation (sequences) for all types
- Ordered foreign keys generation
- Ranges generation
- Idempotent generation by seed number
- devnull output
- CSV output
- Tarantool Column Store output
- Parquet output
- Http output with the ability to configure the format of sent data
- Data partitioning
- Ability to continue generation
- Availability to ignore some models for generation

### Changed

- String templates replaced with jinja like
