# Bag

Bag is a simple, fast and declarative way to create to manage the installed binaries across your system.

## Installation

I recommend to install using Go.

```go
go install github.com/jamesjohnsdev/bag@latest
```

Otherwise, you can download the binary from the [releases](https://github.com/jamesjarvis/bag/releases) page.

## Usage

Bag can be used both globally at a system level, or at a project level.

### Global

A `bag.toml` file and `.bag-lock` is created under `~/.config/bag/`

To install a binary globally, run the following command:

```sh
bag add <remote-path> # e.g. bag add github.com/jamesjohnsdev/issues
```

This will download the binary and add it to your system path.

Optionally, you can specify a version to install:

```sh
bag add <remote-path>@<version> # e.g. bag add github.com/jamesjohnsdev/issues@v0.1.0
```

If you want to install a binary which you have already downloaded, you use the `--local` flag:

```sh
bag add --local <path-to-binary>
```

### Project

Project level installation is useful if you want to install a binary for a specific project, such as development tools.

Before you can use the project level installation, you must initalise the project:

```sh
bag init
```

This will create a `bag.toml` and a `.bag-lock` file in the root of your project.

To install a binary for a project, use the `bag tool` and optional project path:

```sh
bag tool add <remote-path> --project <project-path> # Project-path is optional - defaults to the current directory
```

You can update project tools by running: `bag tool update`, which will update all tools in the project. Otherwise, you can update a specific tool by running: `bag tool update <tool-name>`.

Tools can be removed by running: `bag tool remove <tool-name>`.

> [!NOTE]
> You can substitute `bag tool run` for `bag run` if there is no naming conflict with a globally defined bag.

## Scripts

Scripts are supported using the `--script` flag, and works much the same with the other commands.

Scripts do not support versioning.
They are still pinned to SHA256 hashes, and stored locally.

`commands` are also supported, and will be ran similarly. These are any valid shell syntax.
`commands` cannot be versioned, nor pinned to a specific version.

> [!NOTE]
> You should generally prefer `scripts` over running a command like `cmd = "./script.sh"`.

## How it works

Bag uses a `bag.toml` file to store information about the installed binaries. This file is created when you run `bag init`. It contains a list of binaries and their versions.

It also contains scripts which can be run using the `bag run` command.

E.g.

```toml
[commands]
lint = "golangci-lint run" # Would run with `bag lint` or `bag run lint`

[issues] # Name of binary can be customised however you like
source= "github.com/jamesjohnsdev/issues" # Remote path to the binary
version = "v0.1.0" # Version to install

[something]
version = "v1.23.0"
# Binaries installed using local binary will not have a remote path.

[somethingelse]
source = "https://gist.github.com/be057f2959753ee7c8ab57b3ee6a87ab.git"
type = "script" # Optional - defaults to binary
# Remote or local path to a script works the same as a binary
```

A lock file is also created to ensure versions are consistent using their hash.

Binaries use a global store `~/.local/share/bag/store`:

```
~/.local/share/bag/store
├── issues
│   ├── v0.1.0
│   │   ├── issues
│   │   └── metadata.toml
│   └── v0.2.0
│       ├── issues
│       └── metadata.toml
└── something
    ├── v1.23.0
    │   ├── something
    │   └── metadata.toml
    └── v1.24.0
        ├── something
        └── metadata.toml
```

On download, a SHA256 hash is generated and compared against thelock file.
The store entry is made read-only after installation.

## Shell

Bag provides a `bag shell` command, which will open a shell in a project directory, with all of the local tools available on path.
This provides a way to run the binaries with the correct version, without needing a `bag run` command.

Essentially the purpose of this is to replace the need for a devcontainer or a nix flake.

## Other Commands

```sh
bag sync # Make installed state match `bag.toml` and `.bag-lock`
bag verify # Integrity check
bag clean # Removes unreferenced bags from the store
bag update # Updates declared bags and the lock file
```
