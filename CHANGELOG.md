# Changelog

## [0.1.4](https://github.com/jamesjohnsdev/bag/compare/v0.1.3...v0.1.4) (2026-08-27)


### Features

* support direct binary releases from github ([9b96412](https://github.com/jamesjohnsdev/bag/commit/9b96412015a7c7f063d2022566ac0f5d684803a3))
* support hython separators in release names ([4469bc8](https://github.com/jamesjohnsdev/bag/commit/4469bc8b07df6e824bf31343257dfdf7cd9c88cd))


### Bug Fixes

* enable support for nested archives ([6c180c0](https://github.com/jamesjohnsdev/bag/commit/6c180c04faf22c7fe21756a64995536a25d77302))
* pass correct name for binary installs ([667292d](https://github.com/jamesjohnsdev/bag/commit/667292d2a142168b7335cb13667cb57c663fadc6))
* regression from hython support where system type dropped when cotnaining an underscore ([bef3aea](https://github.com/jamesjohnsdev/bag/commit/bef3aea670b49365233d423243e4da4d8d4d71af))
* remove early close and give ownership to later callers ([554360e](https://github.com/jamesjohnsdev/bag/commit/554360e9b1ed655241d99a52d5c9833fbfbe1f9d))

## [0.1.3](https://github.com/jamesjohnsdev/bag/compare/v0.1.2...v0.1.3) (2026-08-26)


### Features

* handle non-standard casing on runtimes ([69e4ff6](https://github.com/jamesjohnsdev/bag/commit/69e4ff67f1165c8fa4ccba23f2f1c1a1ab877925))

## [0.1.2](https://github.com/jamesjohnsdev/bag/compare/v0.1.1...v0.1.2) (2026-08-26)


### Features

* base provider implementation ([b8b95cd](https://github.com/jamesjohnsdev/bag/commit/b8b95cda909d51e8a677d24e41e3864f3ca617c4))
* config override support ([01444e2](https://github.com/jamesjohnsdev/bag/commit/01444e2492b651cb0787b024ccbb5a65f8d9ee2a))
* github & directURL provider ([41a9179](https://github.com/jamesjohnsdev/bag/commit/41a91793022261cf0d96f0379d1b54be14af75e9))
* github remote installs ([12486b8](https://github.com/jamesjohnsdev/bag/commit/12486b8fda6cb0a7b2433d40e617bde6fb7791e9))
* httpclient shared contstructor ([e2af489](https://github.com/jamesjohnsdev/bag/commit/e2af48974c136b21c57f8b2d2db8780bd2fdb4cd))
* implement github provider ([83f61a7](https://github.com/jamesjohnsdev/bag/commit/83f61a73392edc087c1e6d03fb10be54bce0fabe))
* installation from reader ([0c280e5](https://github.com/jamesjohnsdev/bag/commit/0c280e5a00c16610d5b786b4ed3d8119bee586b0))


### Bug Fixes

* binary name and version not correctly amended after resolution ([c8d6e29](https://github.com/jamesjohnsdev/bag/commit/c8d6e294740f9f2aada1c0ac62fc14bb0c5e1d78))
* check resolved names for traversal risk ([b747501](https://github.com/jamesjohnsdev/bag/commit/b7475013c1a50c1035edb0b3522034385e1cca22))
* correctly parse and handle scheme-less paths ([69bb6b5](https://github.com/jamesjohnsdev/bag/commit/69bb6b509697cfabd580c37fa1df355da8461d3e))
* correctly pass through context through kong bind ([284d1c2](https://github.com/jamesjohnsdev/bag/commit/284d1c29122185d8054f69c4f83d63357fbb1026))
* err checking and golanglint-ci issues ([975f7e7](https://github.com/jamesjohnsdev/bag/commit/975f7e7d9b56577cd94e1f4019244434a377a16b))
* explicitly set `Type` field to avoid potential ambiguity in `add` ([7fa2d81](https://github.com/jamesjohnsdev/bag/commit/7fa2d8130960adc5daa16e31efcaf727ae834319))
* handle misformed config errors ([c9a0f20](https://github.com/jamesjohnsdev/bag/commit/c9a0f2041b7001ce85f3f27857ebfc9749cdacb1))
* ignored err check was segfaulting on no matching system asset ([8862286](https://github.com/jamesjohnsdev/bag/commit/8862286ce57ce205076899ca0fca2f42dfbe958c))
* InstallFromReader now allows reinstall of binary with same metadata ([b10bb79](https://github.com/jamesjohnsdev/bag/commit/b10bb79b25694c146359618dd9d90071efaba665))
* sanitize file name to prevent file transversal risk ([6aff418](https://github.com/jamesjohnsdev/bag/commit/6aff418b26dd36756b744bc90dfcf7a0cb5f8c92))
* sym linking logic moved so it applies to both local and provider options ([71771fe](https://github.com/jamesjohnsdev/bag/commit/71771fe3458d56b20268f0b0ee99e9b90a800b41))

## [0.1.1](https://github.com/jamesjohnsdev/bag/compare/v0.1.0...v0.1.1) (2026-08-24)


### Features

* add locally available binaries to bag ([c93fea3](https://github.com/jamesjohnsdev/bag/commit/c93fea39084d1563cd9985d36b6771a3f42e4d40))
* better error descriptors ([0e96e86](https://github.com/jamesjohnsdev/bag/commit/0e96e86db7a125d94e7dcde5d3cbfa9416bdf3c3))
* find bag.toml ([8263efd](https://github.com/jamesjohnsdev/bag/commit/8263efd30578830d96dc2b13380fd53417dc685a))
* init command ([db7c339](https://github.com/jamesjohnsdev/bag/commit/db7c3397761022d6ad00ae75cfecebc313f443cd))
* lockfile creation and parsing ([df2106c](https://github.com/jamesjohnsdev/bag/commit/df2106c82fda21ea7b250de29b4cf5e577ab681f))
* manifest parsing and writing ([c36f3a2](https://github.com/jamesjohnsdev/bag/commit/c36f3a2270690c6e722c78cffb941cb75bef5d6e))
* store + local install ([9fbdd5f](https://github.com/jamesjohnsdev/bag/commit/9fbdd5f1b4ebf5772163702955c5d2531b222d0c))
* sym linking of binaries onto path ([e12843a](https://github.com/jamesjohnsdev/bag/commit/e12843aa2998c454bd630b3a8c8eebdc0bc12ce2))


### Bug Fixes

* add main.go stub so lint/CodeQL have source to analyze ([348e132](https://github.com/jamesjohnsdev/bag/commit/348e132358c8fe6bda7bfdc8bbf2d3220821b8d4))
* FindManifest allows choosing local or global target ([6c2702a](https://github.com/jamesjohnsdev/bag/commit/6c2702ae6906208b0b1c2d420559ae7cdf4ffed3))
