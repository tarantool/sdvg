# User Guide

## Table of Contents

- [Configuration](#configuration)
  - [SDVG instance configuration](#sdvg-instance-configuration)
    - [Description of SDVG instance configuration fields](#description-of-sdvg-instance-configuration-fields)
    - [Examples of SDVG instance configuration](#examples-of-sdvg-instance-configuration)
  - [Data generation configuration](#data-generation-configuration)
    - [Description of data generation configuration fields](#description-of-data-generation-configuration-fields)
    - [Examples of data generation configuration](#examples-of-data-generation-configuration)
- [Launch](#launch)
  - [Data generation](#data-generation)
  - [Ignoring conflicts](#ignoring-conflicts)
  - [Continuing generation](#continuing-generation)

## Configuration

SDVG uses two configuration files: the SDVG instance configuration file and the data generation configuration file.

### SDVG instance configuration

#### Description of SDVG instance configuration fields

The SDVG instance configuration includes the following fields:

- `log_format`: Log format.
  Supported values: `text`, `json`. Default is `text`.
- `http`: HTTP server configuration described by the `HTTPConfig` structure.
- `open_ai`: OpenAI configuration described by the `OpenAI` structure.

The `http` structure describes the HTTP server configuration used for
interacting with SDVG and contains the following fields:

- `listen_address`: Address for the HTTP server to listen on. Default is `:8080`.
- `read_timeout`: Data read timeout. Default is `1m` (1 minute).
- `write_timeout`: Data write timeout. Default is `1m` (1 minute).
- `idle_timeout`: Idle timeout. Default is `1m` (1 minute).

The `open_ai` structure describes the OpenAI configuration and includes the following fields:

- `api_key`: API key for accessing OpenAI.
- `base_url`: Base URL for the OpenAI API.
- `model`: OpenAI model.

#### Examples of SDVG instance configuration

Example configuration for HTTP server:

```yaml
http:
  listen_address: localhost:8080
  read_timeout: 1m
  write_timeout: 30s
  idle_timeout: 1m30s
```

Example configuration for OpenAI:

```yaml
open_ai:
  api_key: "sk-123"
  base_url: "http://10.0.1.100:11434/v1"
  model: "deepseek-r1:70b-llama-distill-q8_0"
```

### Data generation configuration

This configuration is directly used for data generation after launching SDVG.

#### Description of data generation configuration fields

The data generation configuration includes the following fields:

- `random_seed`: Seed for random number generation. If omitted or set to `0`, a random value is used.
- `workers_count`: Number of threads for data generation. Defaults to CPU count multiplied by 4.
- `batch_size`: Batch size for data generation and output. Default is `1000`.
- `models`: Map of data models, with the key as the model name and the value as `models[*]` structure.
- `models_to_ignore`: List of models to exclude from this SDVG data generation run
  (foreign keys referencing these models will still work if `random_seed` and `rows_count` remain unchanged).
- `output`: Output configuration for generated data, described by the `output` structure.

The `output` structure describes the generated data output configuration:

- `type`: Output format type. Supported values: `devnull`, `csv`, `parquet`, `http`, `tcs`. Default is `csv`.
- `dir`: Directory for storing generated data. Default is `./output`.
- `create_model_dir`: Specifies whether separate directories are created for each model. Default is `false`.
- `params`: Parameters for the chosen output format type, described by the `output.params` structure.
- `checkpoint_interval`: Frequency of progress checkpoint file updates. Default is `5s` (5 seconds).

The `models[*]` structure describes a data generation model and includes:

- `rows_count`: Number of rows to generate. Required field.
- `rows_per_file`: Number of rows per file, supported by `csv` and `parquet`. Defaults is `rows_count`.
- `generate_from`: Starting row number for generation. Default is `0`.
- `generate_to`: Ending row number for generation. Default is `rows_count`.
- `model_dir`: Directory to store data for this model, relative to `output_dir`. Defaults to model name.
- `columns`: List of columns described by the `models[*].columns` structure.
- `partition_columns`: Columns used for data partitioning. Supported only for `parquet`.

The `models[*].partition_columns` structure specifies data partitioning columns:

- `name`: Column name from schema `models[*].columns`. Required field.
- `write_to_output`: Flag indicating whether the partition column is included in final data files.

The `models[*].columns` structure describes a column in a data model:

- `name`: Column name. Required field.
- `type`: Column data type. Supported values: `integer`, `float`, `string`, `datetime`, `uuid`.
- `type_params`: Parameters for the chosen data type (`models[*].columns[*].type_params` structure).
- `values`: Enumeration of possible column values. Cannot coexist with `distinct` parameters.
- `ordered`: Indicates if column values should be ordered (similar to sequence).
- `distinct_percentage`: Percentage of unique values. Must be between `0` and `1`. Cannot coexist with `distinct_count`.
- `distinct_count`: Number of unique values. Must be greater than `0`. Cannot coexist with `distinct_percentage`.
- `null_percentage`: Percentage of null values. Must be between `0` and `1`.
- `ranges`: A set of parameter ranges for a column that allows you to specify several configurations
  with their percentage distribution (`range_percentage`). Each range (`ranges[*]`) can contain:
  - `type_params`: Parameters for the selected data type.
  - `values`: Enumeration of possible values in the range.
  - `ordered`: Flag for ordered values.
  - `distinct_percentage`: Percentage of unique values.
  - `distinct_count`: Number of unique values.
  - `null_percentage`: Percentage of null values.
  - `range_percentage`: Percentage of this range relative to total data.
- `parquet_params`: Parameters for formatting values in `parquet` output.
- `foreign_key`: Foreign key reference in the format `model_name.column_name`. Values are sourced from this column.
  Cannot coexist with other column parameters.
- `foreign_key_order`: Indicates if the foreign key order should be preserved.
  Useful for maintaining value correspondence with external tables.

> **Attention**: The `ranges` parameter and direct specification of parameters at the column level
> (`values`, `type_params`, `distinct_percentage`, `distinct_count`, `null_percentage`, `ordered`)
> are mutually exclusive. They cannot be used simultaneously.

Structure `models[*].columns[*].parquet_params`:

- `encoding`: Encoding for the column. Supported values: `PLAIN`, `RLE_DICTIONARY`, `DELTA_BINARY_PACKED`,
  `DELTA_BYTE_ARRAY`, `DELTA_LENGTH_BYTE_ARRAY`. Default is `PLAIN`.

Structure `models[*].columns[*].type_params` for data type `integer`:

- `bit_width`: Bit width for integer. Supported values: `8`, `16`, `32`, `64`. Default is `32`.
- `from`: Minimum value for integer. Defaults to the minimum possible value for the selected bit width.
- `to`: Maximum value for integer. Defaults to the maximum possible value for the selected bit width.

Structure `models[*].columns[*].type_params` for data type `float`:

- `bit_width`: Bit width for float. Supported values: `32`, `64`. Default is `32`.
- `from`: Minimum value for float. Defaults to the minimum possible value for the selected bit width.
- `to`: Maximum value for float. Defaults to the maximum possible value for the selected bit width.

Structure `models[*].columns[*].type_params` for data type `string`:

- `min_length`: Minimum string length. Default is `1`.
- `max_length`: Maximum string length. Default is `32`.
- `logical_type`: Logical type of string. Supported values: `first_name`, `last_name`, `phone`, `text`.
- `template`: Template for string generation. Allows you to use the values of any columns of the generated model.
  Information about the functions available in template strings is described at the end of this section.
  Cannot coexist with `ordered`, `distinct_percentage` or `distinct_count`.
- `pattern`: Pattern for string generation. The `A` symbol is any capital letter, the `a` symbol is any small letter,
  symbol `0` is any digit, the `#` symbol is any character, and the other characters remain as they are.
- `locale`: Locale for generated strings. Supported values: `ru`, `en`. Default is `en`.
- `without_large_letters`: Flag indicating if uppercase letters should be excluded from the string.
- `without_small_letters`: Flag indicating if lowercase letters should be excluded from the string.
- `without_numbers`: Flag indicating if numbers should be excluded from the string.
- `without_special_chars`: Flag indicating if special characters should be excluded from the string.

Structure `models[*].columns[*].type_params` for data type `datetime`:

- `from`: Minimum date-time value. Default is `01.01.1900`.
- `to`: Maximum date-time value. Default is `01.01.2025`.

Structure `output.params` for format `csv`:

- `float_precision`: Floating-point number precision. Default is `2`.
- `datetime_format`: Date-time format. Default is `2006-01-02T15:04:05Z07:00`.
- `without_headers`: Flag indicating if CSV headers should be excluded from data files.
- `delimiter`: Single-character CSV delimiter. Default is `,`.

Structure `output.params` for format `parquet`:

- `compression_codec`: Compression codec. Supported values: `UNCOMPRESSED`, `SNAPPY`, `GZIP`, `LZ4`, `ZSTD`.
  Default is `UNCOMPRESSED`.
- `float_precision`: Floating-point number precision. Default is `2`.
- `datetime_format`: Date-time format. Supported values: `millis`, `micros`. Default is `millis`.

Structure `output.params` for format `http`:

- `endpoint`: Endpoint for sending data.
- `timeout`: Timeout for sending data, specified as a string combining `h`, `m`, `s` without spaces, e.g.,
  `1h`, `5m30s`, `2h5s`. Default is `1m` (1 minute).
- `batch_size`: Number of data records sent in one request. Default is `1000`.
- `workers_count`: Number of threads for writing data. Default is `1`. *Experimental field.*
- `headers`: HTTP request headers specified as a dictionary. Default is none.
- `format_template`: Template-based format for sending data, configured using Golang templates.  
  There are 2 fields available for use in `format_template`:
    * `ModelName` - name of the model.
    * `Rows` - array of records, where each element is a dictionary representing a data row.
      Dictionary keys correspond to column names, and values correspond to data in those columns.

  You can read about the available functions and the use of template strings at the end of this section.

  Example value for the `format_template` field:

  ```yaml
  format_template: |
    {
      "table_name": "{{ .ModelName }}",
      "meta": {
        "rows_count": {{ len .Rows }}
      },
      "rows": [
        {{- range $i, $row := .Rows }}
          {{- if $i}},{{ end }}
          {
            "id": {{ index $row "id" }},
            "username": "{{ index $row "name" }}"
          }
        {{- end }}
      ]
    }
  ```

  Default value for the `format_template` field:

  ```yaml
  format_template: |
    {
      "table_name": {{ .ModelName }},
      "rows": {{ json .Rows }}
    }
  ```

Structure of `output.params` for `tcs` format:

Similar to the structure for the `http` format,
except that the `format_template` field is immutable and always set to its default value.

Using Template Strings::

Template strings are implemented using the standard golang library, you can read about
all its features and available functions in this [documentation](https://pkg.go.dev/text/template).

Accessing Data:

In a template, data is accessed using `.`(the object or value passed to the template)
and the field name, for example: `{{ .var }}`.

Function calls:

- direct call: `{{ upper .name }}`.
- using pipe: `{{ .name | upper }}`.

In addition to standard functions, the project provides `4` custom functions:

- `upper`: converts the string to upper case.
- `lower`: converts the string to lower case.
- `len`: returns the length of the element.
- `json`: converts the element to a JSON string.

Usage restrictions:

The `lower`, and `upper` functions are available only in the `template` field of the `string` data type.
The `len` and `json` functions are available only in the `format_template` field of the output parameters.

#### Examples of data generation configuration

Example data model configuration:

```yaml
workers_count: 32
batch_size: 1000
random_seed: 0
output:
  type: "devnull"
  dir: output-dir
models:
  token:
    rows_count: 500000
    model_dir: token_model
    columns:
      - name: id
        type: uuid
      - name: user_id
        foreign_key: user.id
      - name: session_id
        type: string
        type_params:
          min_length: 16
          max_length: 32
        distinct_percentage: 1
      - name: token_type
        type: string
        values:
          - "access"
          - "refresh"
  user:
    rows_count: 10000
    columns:
      - name: id
        type: integer
        type_params:
          from: 1
          to: 500000
        ordered: true
      - name: str_id
        type: string
        ordered: true
      - name: ru_phone
        type: string
        type_params:
          logical_type: phone
          locale: ru
      - name: first_name_ru
        type: string
        type_params:
          logical_type: first_name
          locale: ru
      - name: last_name_ru
        type: string
        type_params:
          logical_type: last_name
          locale: ru
      - name: first_name_en
        type: string
        type_params:
          logical_type: first_name
      - name: passport
        type: string
        type_params:
          pattern: AA 00 000 000
        distinct_percentage: 1
        ordered: true
      - name: email
        type: string
        type_params:
          template: "{{ .first_name_en | lower }}.{{ .id }}@example.com"
      - name: rating
        type: float
        type_params:
          from: 0.0
          to: 5.0
      - name: created
        type: datetime
        type_params:
          from: 2020-01-01T00:00:00Z
        ordered: true
      - name: birthday
        type: datetime
        ranges:
          - type_params:
              from: 1900-01-01T00:00:00Z
          - values: [null]
            range_percentage: 0.1
          - values:
              - 2005-03-09T04:44:00Z
```

Example configuration for generating CSV files:

```yaml
output:
  type: csv
  params:
    float_precision: 1
    datetime_format: 2006-01-02
models:
  user:
    rows_count: 10000
    columns:
      - name: id
        type: uuid
      - name: session_id
        type: string
```

Example configuration for generating Parquet files:

```yaml
output:
  type: parquet
  params:
    float_precision: 1
    datetime_format: millis
    compression_codec: UNCOMPRESSED
models:
  token:
    rows_count: 500000
    rows_per_file: 250000
    columns:
      - name: id
        type: uuid
      - name: session_id
        type: string
        parquet:
          encoding: RLE_DICTIONARY
        distinct_percentage: 1
```

Example configuration for sending generated data via HTTP:

```yaml
output:
  type: http
  params:
    endpoint: "http://127.0.0.1:8080/insert"
    timeout: 30s
    headers:
      Authorization: "Bearer <token>"
    format_template: |
      {
        "table_name": "{{ .ModelName }}",
        "meta": {
          "rows_count": {{ len .Rows }}
        },
        "rows": {{ json .Rows }}
      }

models:
  user:
    rows_count: 10000
    columns:
      - name: id
        type: uuid
      - name: session_id
        type: string
```

Example configuration for sending generated data to TCS:

```yaml
output:
  type: tcs
  params:
    endpoint: "http://127.0.0.1:7101/insert"
    timeout: 30s
models:
  user:
    rows_count: 10000
    columns:
      - name: id
        type: uuid
      - name: session_id
        type: string
```

## Launch

To start in interactive mode, simply run the SDVG binary:

```shell
sdvg
```

To get information about available commands and their arguments:

```shell
sdvg -h
sdvg --help
sdvg generate -h
```

### Data generation

Before starting data generation, SDVG checks the output directory for conflicting files.
If conflicts are found, they will be displayed as a list of errors upon startup.
This helps avoid overwriting or corrupting existing data.

To start data generation with a specified configuration file:

```shell
sdvg generate ./models.yml
```

### Ignoring conflicts

If you want to automatically remove conflicting files from the output directory
and continue generation without additional prompts, use the `-F` or `--force` flag:

```shell
sdvg generate --force ./models.yml
```

### Continuing generation

To continue generation from the last recorded row:

```shell
sdvg generate --continue-generation ./models.yml
```

> **Important**: To correctly continue generation, you must not change the generation configuration
> or already generated data.
