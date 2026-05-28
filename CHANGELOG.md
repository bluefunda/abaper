# Changelog

> **Note:** Prior to v2.0.0 this project was published as `github.com/bluefunda/abaper-cli`. Historical links in this changelog point to the old repository for reference.

## [1.5.0](https://github.com/bluefunda/abaper-cli/compare/v1.4.4...v1.5.0) (2026-05-13)


### Features

* add macOS notarized brew cask installer ([abaa043](https://github.com/bluefunda/abaper-cli/commit/abaa04326c7d6e70ab08c5319e3427a160dceb04))

## [1.4.4](https://github.com/bluefunda/abaper-cli/compare/v1.4.3...v1.4.4) (2026-04-24)


### Bug Fixes

* use DOCKER_USERNAME/DOCKER_PASSWORD org secrets, add continue-on-error to description step ([c10ddcb](https://github.com/bluefunda/abaper-cli/commit/c10ddcbbe1e7a3be1ad9967867f8030105a49356))

## [1.4.3](https://github.com/bluefunda/abaper-cli/compare/v1.4.2...v1.4.3) (2026-04-24)


### Bug Fixes

* fold dockerhub-readme into docker job to fix 403 on description update ([e8f5513](https://github.com/bluefunda/abaper-cli/commit/e8f55137963f2990d08a5461325058604f3adb8e))

## [1.4.2](https://github.com/bluefunda/abaper-cli/compare/v1.4.1...v1.4.2) (2026-04-23)


### Bug Fixes

* replace fi with done in install.sh dependency check loop ([d2a3d2e](https://github.com/bluefunda/abaper-cli/commit/d2a3d2ef446646bbfd8ff8450efd0a72ca48f9df))

## [1.4.1](https://github.com/bluefunda/abaper-cli/compare/v1.4.0...v1.4.1) (2026-04-23)


### Bug Fixes

* correct docker image name from abaper to abaper-cli ([#22](https://github.com/bluefunda/abaper-cli/issues/22)) ([161c1e0](https://github.com/bluefunda/abaper-cli/commit/161c1e05e436b4524e89985e94a09becade4a242))

## [1.4.0](https://github.com/bluefunda/abaper-cli/compare/v1.3.1...v1.4.0) (2026-04-22)


### Features

* add install.sh for one-line installation ([#20](https://github.com/bluefunda/abaper-cli/issues/20)) ([dbd853e](https://github.com/bluefunda/abaper-cli/commit/dbd853ebc8d58865090a47a3363ff394e53ed5b9))

## [1.3.1](https://github.com/bluefunda/abaper-cli/compare/v1.3.0...v1.3.1) (2026-04-02)


### Bug Fixes

* revert auth host to auth.bluefunda.com ([#16](https://github.com/bluefunda/abaper-cli/issues/16)) ([b64205e](https://github.com/bluefunda/abaper-cli/commit/b64205e5b8965a1d527a330f2670ed460f835a76))

## [1.3.0](https://github.com/bluefunda/abaper-cli/compare/v1.2.1...v1.3.0) (2026-04-02)


### Features

* add telemetry, signup command, route login through bluefunda.com ([#14](https://github.com/bluefunda/abaper-cli/issues/14)) ([3fe4c64](https://github.com/bluefunda/abaper-cli/commit/3fe4c646c3ec4ecb2f396669a4801f29e8883274))

## [1.2.1](https://github.com/bluefunda/abaper-cli/compare/v1.2.0...v1.2.1) (2026-03-13)


### Bug Fixes

* switch from Homebrew cask to formula to avoid macOS Gatekeeper ([#11](https://github.com/bluefunda/abaper-cli/issues/11)) ([cd07ae7](https://github.com/bluefunda/abaper-cli/commit/cd07ae7a17ecf9e907ae40420e94f6654e1a0b94))

## [1.2.0](https://github.com/bluefunda/abaper-cli/compare/v1.1.1...v1.2.0) (2026-03-13)


### Features

* add AI chat, unit tests, list, and missing gateway API coverage ([#8](https://github.com/bluefunda/abaper-cli/issues/8)) ([96f6436](https://github.com/bluefunda/abaper-cli/commit/96f643628173bd9c83bff085c78c69b09390e190))


### Bug Fixes

* use RELEASE_PAT secret for release-please workflow ([#9](https://github.com/bluefunda/abaper-cli/issues/9)) ([62be342](https://github.com/bluefunda/abaper-cli/commit/62be34238745ecf1a00817d5cfc492062ba29a99))

## [1.1.1](https://github.com/bluefunda/abaper-cli/compare/v1.1.0...v1.1.1) (2026-03-11)


### Bug Fixes

* use homebrew_casks, HOMEBREW_TAP_TOKEN, and correct DOCKERHUB_TOKEN ([#6](https://github.com/bluefunda/abaper-cli/issues/6)) ([7dc3984](https://github.com/bluefunda/abaper-cli/commit/7dc39846a9555bbc02081b84c316fcb33448bb51))

## [1.1.0](https://github.com/bluefunda/abaper-cli/compare/v1.0.1...v1.1.0) (2026-03-11)


### Features

* initial ABAPer CLI project ([35d3df0](https://github.com/bluefunda/abaper-cli/commit/35d3df0e22c22821f68502c2c11d03913034486c))
* rename binaries to abaper, add man page and Homebrew tap ([#4](https://github.com/bluefunda/abaper-cli/issues/4)) ([748c629](https://github.com/bluefunda/abaper-cli/commit/748c62933a75dfee8a715743f11472fc03805bdb))


### Bug Fixes

* align workflow files with release-foundry patterns ([a86b811](https://github.com/bluefunda/abaper-cli/commit/a86b811e66e81085b5f9e5b7c9beb60dac23570f))
* handle resp.Body.Close() error returns for errcheck lint ([2ec203a](https://github.com/bluefunda/abaper-cli/commit/2ec203af8cd662b4a8065c772cb1cd114f0a5ab1))
* inline CI and release workflows ([59c6b24](https://github.com/bluefunda/abaper-cli/commit/59c6b242e2c4d76a6130eeac305d8eebaccef338))
* push Docker image to bluefunda/abaper and add manual deploy workflow ([5b042c1](https://github.com/bluefunda/abaper-cli/commit/5b042c1db6680093b515f7106e02778fc0831881))
* use GH_PAT for release-please to trigger CI on release PRs ([2645d03](https://github.com/bluefunda/abaper-cli/commit/2645d03212bc0a58a49e56435efcb511c40d302c))

## [1.0.1](https://github.com/bluefunda/abaper-cli/compare/v1.0.0...v1.0.1) (2026-03-11)


### Bug Fixes

* push Docker image to bluefunda/abaper and add manual deploy workflow ([5b042c1](https://github.com/bluefunda/abaper-cli/commit/5b042c1db6680093b515f7106e02778fc0831881))

## 1.0.0 (2026-03-11)


### Features

* initial ABAPer CLI project ([35d3df0](https://github.com/bluefunda/abaper-cli/commit/35d3df0e22c22821f68502c2c11d03913034486c))


### Bug Fixes

* align workflow files with release-foundry patterns ([a86b811](https://github.com/bluefunda/abaper-cli/commit/a86b811e66e81085b5f9e5b7c9beb60dac23570f))
* handle resp.Body.Close() error returns for errcheck lint ([2ec203a](https://github.com/bluefunda/abaper-cli/commit/2ec203af8cd662b4a8065c772cb1cd114f0a5ab1))
* inline CI and release workflows ([59c6b24](https://github.com/bluefunda/abaper-cli/commit/59c6b242e2c4d76a6130eeac305d8eebaccef338))
* use GH_PAT for release-please to trigger CI on release PRs ([2645d03](https://github.com/bluefunda/abaper-cli/commit/2645d03212bc0a58a49e56435efcb511c40d302c))
