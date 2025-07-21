# Synthetic Data Values Generator (SDVG)

[![CI][actions-badge]][actions-url]
[![Release][release-badge]][release-url]
[![Coverage Status][test-coverage-badge]][test-coverage-url]

[actions-badge]: https://img.shields.io/github/check-runs/tarantool/sdvg/master
[actions-url]: https://github.com/tarantool/sdvg/actions
[release-badge]: https://img.shields.io/github/v/release/tarantool/sdvg?filter=!latest
[release-url]: https://github.com/tarantool/sdvg/releases
[test-coverage-badge]: https://img.shields.io/coverallsCoverage/github/tarantool/sdvg?branch=master
[test-coverage-url]: https://coveralls.io/github/tarantool/sdvg?branch=master

## Язык

- [English](README.md)
- **Русский**

## Описание продукта

SDVG (Synthetic Data Values Generator) — это инструмент для генерации синтетических данных.
Он поддерживает различные форматы запуска, типы данных для генерации и форматы вывода.

Форматы запуска:

- CLI - генерация данных, создание конфигураций и их валидация через консоль;
- HTTP сервер - принимает запросы на генерацию по HTTP API и отправляет/сохраняет их в указанное место.

Типы данных:

- строки (английские, русские);
- целые и вещественные числа;
- даты со временем;
- UUID.

Типы строк:

- случайные;
- тексты;
- имена;
- фамилии;
- телефонные номера;
- шаблоны.

Каждый из типов данных можно генерировать со следующими опциями:

- указание процента/количества уникальных значений на колонку;
- упорядоченная генерация (sequence);
- указание внешнего ключа;
- идемпотентная генерация по seed числу;
- генерация значений из диапазонов с процентным распределением значений.

Форматы вывода:

- devnull;
- CSV файлы;
- Parquet файлы;
- HTTP API;
- Tarantool Column Store HTTP API.

## Быстрый старт

Пример модели данных, которая генерирует 10 000 строк пользователей и записывает их в CSV-файл:

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

Сохраните это в файл `simple_model.yml`, затем выполните:

```bash
./sdvg generate simple_model.yml
```

Это создаст CSV-файл с фейковыми пользовательскими данными, такими как `id` и `name`:

```csv
id,name
c8a53cfd-1089-4154-9627-560fbbea2fef,Sutherlan
b5c024f8-3f6f-43d3-b021-0bb2305cc680,Hilton
5adf8218-7b53-41bb-873d-c5768ca6afa2,Craggy
...
```

Чтобы запустить генератор в интерактивном режиме:

```bash
./sdvg
```

Чтобы посмотреть доступные команды и аргументы:

```bash
./sdvg -h
./sdvg --help
./sdvg generate -h
```

Больше информации можно найти в [руководстве по эксплуатации](./doc/ru/usage.md).

## Документация

- [Руководство по эксплуатации](./doc/ru/usage.md)
- [Руководство для разработчиков](./doc/ru/contributing.md)
- [Цели и соответствие стандартам](./doc/ru/overview.md)
- [Список изменений](./CHANGELOG.md)
- [Лицензия](./LICENSE)

## Разработчики

- [@hackallcode](https://github.com/hackallcode)
- [@ReverseTM](https://github.com/ReverseTM)
- [@choseenonee](https://github.com/choseenonee)
- [@Hoodie-Huuuuu](https://github.com/Hoodie-Huuuuu)
