# Manifest Files and Lockfiles Reference

This document lists common manifest files and their corresponding lockfiles across different programming languages and package managers.

- Some ecosystems don't have standardized lockfiles
- Lockfiles ensure reproducible builds by recording exact versions
- Some tools use the same file as both manifest and lockfile (with version pinning)
- Monorepos may have multiple manifest/lockfile pairs

## JavaScript / Node.js
Mapping:
> `package.json` -> `package-lock.json`, `npm-shrinkwrap.json`, `yarn.lock`, `pnpm-lock.yaml`, `bun.lock`, `deno.lock`
### npm
- **Manifest**: `package.json`
- **Lockfile**: `package-lock.json`, `npm-shrinkwrap.json`
- https://docs.npmjs.com/cli/v6/configuring-npm/package-lock-json
- https://docs.npmjs.com/cli/v6/configuring-npm/shrinkwrap-json

### Yarn (Classic & Berry)
- **Manifest**: `package.json`
- **Lockfile**: `yarn.lock`
- https://classic.yarnpkg.com/lang/en/docs/yarn-lock/

### pnpm
- **Manifest**: `package.json`
- **Lockfile**: `pnpm-lock.yaml`
- https://github.com/pnpm/spec/tree/master/lockfile

### Bun
- **Manifest**: `package.json`
- **Lockfile**: `bun.lockb` (deprecated), `bun.lock`

### Deno
- **Manifest**: `package.json`, `deno.json`
- **Lockfile**: `deno.lock`
- https://docs.deno.com/runtime/fundamentals/configuration/#lockfile



## Python

### pip
- **Manifest**: `requirements.txt`, `setup.py`, `pyproject.toml`
- **Lockfile**: `requirements.txt` (can serve as both when pinned)

### Pipenv
- **Manifest**: `Pipfile`
- **Lockfile**: `Pipfile.lock`

### Poetry
- **Manifest**: `pyproject.toml`
- **Lockfile**: `poetry.lock`

### PDM
- **Manifest**: `pyproject.toml`
- **Lockfile**: `pdm.lock`

### uv
- **Manifest**: `pyproject.toml`
- **Lockfile**: `uv.lock`

## Ruby

### Bundler
- **Manifest**: `Gemfile`
- **Lockfile**: `Gemfile.lock`

## PHP

### Composer
- **Manifest**: `composer.json`
- **Lockfile**: `composer.lock`

## Rust

### Cargo
- **Manifest**: `Cargo.toml`
- **Lockfile**: `Cargo.lock`

## Go

### Go Modules
- **Manifest**: `go.mod`
- **Lockfile**: `go.sum`

## Java

### Maven
- **Manifest**: `pom.xml`
- **Lockfile**: Not standard, but some use Maven lockfile plugin or `maven.lockfile`

### Gradle
- **Manifest**: `build.gradle`, `build.gradle.kts`, `settings.gradle`, `settings.gradle.kts`
- **Lockfile**: `gradle.lockfile`, `*.lockfile` (per configuration)

## C# / .NET

### NuGet
- **Manifest**: `*.csproj`, `*.fsproj`, `*.vbproj`, `packages.config`
- **Lockfile**: `packages.lock.json`

## Swift

### Swift Package Manager
- **Manifest**: `Package.swift`
- **Lockfile**: `Package.resolved`

## Dart / Flutter

### Pub
- **Manifest**: `pubspec.yaml`
- **Lockfile**: `pubspec.lock`

## Elixir

### Mix
- **Manifest**: `mix.exs`
- **Lockfile**: `mix.lock`

## Scala

### sbt
- **Manifest**: `build.sbt`, `project/build.properties`, `project/plugins.sbt`
- **Lockfile**: Not standard, but coursier can generate `coursier.lock`

## Clojure

### Leiningen
- **Manifest**: `project.clj`
- **Lockfile**: Not standard

### tools.deps
- **Manifest**: `deps.edn`
- **Lockfile**: Not standard

## R

### CRAN
- **Manifest**: `DESCRIPTION`
- **Lockfile**: Not standard

### renv
- **Manifest**: `DESCRIPTION`
- **Lockfile**: `renv.lock`

## Perl

### cpanm
- **Manifest**: `cpanfile`
- **Lockfile**: `cpanfile.snapshot`

## OCaml

### opam
- **Manifest**: `*.opam`, `dune-project`
- **Lockfile**: `*.opam.locked`

## Haskell

### Stack
- **Manifest**: `stack.yaml`, `package.yaml`
- **Lockfile**: `stack.yaml.lock`

### Cabal
- **Manifest**: `*.cabal`, `cabal.project`
- **Lockfile**: `cabal.project.freeze`

## Kotlin

### Gradle (Kotlin DSL)
- **Manifest**: `build.gradle.kts`, `settings.gradle.kts`
- **Lockfile**: `gradle.lockfile`, `*.lockfile`

## Zig

### Zig Build System
- **Manifest**: `build.zig.zon`
- **Lockfile**: Not standard in current versions

## C / C++

### Conan
- **Manifest**: `conanfile.txt`, `conanfile.py`
- **Lockfile**: `conan.lock`

### vcpkg
- **Manifest**: `vcpkg.json`
- **Lockfile**: `vcpkg-lock.json`

## Nim

### Nimble
- **Manifest**: `*.nimble`
- **Lockfile**: `nimble.lock`

## Crystal

### Shards
- **Manifest**: `shard.yml`
- **Lockfile**: `shard.lock`

## Lua

### LuaRocks
- **Manifest**: `*.rockspec`
- **Lockfile**: `luarocks.lock` (not standard, tool-dependent)

