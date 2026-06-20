# Repository Guidelines

## Project Structure & Module Organization

This repository contains a small Go CLI for generating RSS 2.0 XML from HTML pages. The main application code is in `main.go`. Unit tests live beside the source in `main_test.go`. Example configuration is kept in `config.example.yml`. Planning notes are under `docs/plans/`.

Generated files such as `feed.xml` and local build artifacts such as the `rssgen` binary should not be committed.

## Build, Test, and Development Commands

Use these commands from the repository root:

```sh
go run . -config config.example.yml
```

Runs the CLI and writes RSS XML to standard output.

```sh
go run . -config config.example.yml -output feed.xml
```

Runs the CLI and writes RSS XML to `feed.xml`.

```sh
go test ./...
```

Runs all Go tests.

```sh
go build .
```

Builds the local CLI binary.

## Coding Style & Naming Conventions

Format Go code with `gofmt` before committing:

```sh
gofmt -w main.go main_test.go
```

Use idiomatic Go naming: exported identifiers use `CamelCase`, unexported identifiers use `camelCase`, and tests use `TestName` functions. Keep parsing, fetching, extraction, and RSS rendering logic in focused functions so they remain testable.

## Testing Guidelines

Tests use Go's standard `testing` package. Add or update tests in `main_test.go` when changing configuration parsing, XPath extraction, URL resolution, or RSS generation. Prefer small HTML fixtures in tests rather than network access. Network checks are useful manually, but unit tests should remain deterministic.

## Configuration Notes

Configuration is YAML. `settings.channel.link` controls the RSS channel link. Each item under `feeds` must include `title`, `url`, and `xpath`.

## Commit & Pull Request Guidelines

Recent history uses concise commit subjects, for example `RSS 生成プログラムを追加` and `add .gitignore`. Keep commit titles short and specific. Pull requests should describe the behavior change, include test results such as `go test ./...`, and mention any changes to the YAML configuration format.
