# Building Wave Terminal

These instructions are for setting up dependencies and building Wave Terminal from source on macOS, Linux, and Windows.

## Prerequisites

### OS-specific dependencies

See [Minimum requirements](README.md#minimum-requirements) to learn whether your OS is supported.

#### macOS

macOS does not have any platform-specific dependencies.

#### Linux

You must have `zip` installed. We also require the [Zig](https://ziglang.org/) compiler for statically linking CGO.

Debian/Ubuntu:

```sh
sudo apt install zip snapd
sudo snap install zig --classic --beta
```

Fedora/RHEL:

```sh
sudo dnf install zip zig
```

Arch:

```sh
sudo pacman -S zip zig
```

##### For packaging

For packaging, the following additional packages are required:

- `fpm` &mdash; If you're on x64 you can skip this. If you're on ARM64, install fpm via [Gem](https://rubygems.org/gems/fpm)
- `rpm` &mdash; If you're not on Fedora, install RPM via your package manager.
- `snapd` &mdash; If your distro doesn't already include it, [install `snapd`](https://snapcraft.io/docs/installing-snapd)
- `lxd` &mdash; [Installation instructions](https://canonical.com/lxd/install)
- `snapcraft` &mdash; Run `sudo snap install snapcraft --classic`
- `libarchive-tools` &mdash; Install via your package manager
- `binutils` &mdash; Install via your package manager
- `libopenjp2-tools` &mdash; Install via your package manager
- `squashfs-tools` &mdash; Install via your package manager

#### Windows

You will need the [Zig](https://ziglang.org/) compiler for statically linking CGO.

You can find installation instructions for Zig on Windows [here](https://ziglang.org/learn/getting-started/#managers).

### Task

Download and install Task (to run the build commands): https://taskfile.dev/installation/

Task is a modern equivalent to GNU Make. We use it to coordinate our build steps. You can find our full Task configuration in [Taskfile.yml](Taskfile.yml).

### Go

Download and install Go via your package manager or directly from the website: https://go.dev/doc/install

### NodeJS

Make sure you have a NodeJS 22 LTS installed.

See NodeJS's website for platform-specific instructions: https://nodejs.org/en/download

### Yarn Modern

Once you have NodeJS installed, you'll need to enable Corepack so that Yarn Modern can work:

```sh
corepack enable
```

If your NodeJS distribution does not ship with Corepack, you can install it manually using NPM:

```sh
npm install -g corepack
corepack enable
```

Corepack's official documentation says to uninstall Yarn and PnPM first, but this is probably overkill. If you have any issues, try removing them from your system before continuing.

For more information on Corepack, check out [this link](https://yarnpkg.com/corepack).

## Clone the Repo

```sh
git clone git@github.com:wavetermdev/waveterm.git
```

or

```sh
git clone https://github.com/wavetermdev/waveterm.git
```

## Install code dependencies

The first time you clone the repo, you'll need to run the following to load the dependencies. If you ever have issues building the app, try running this again:

```sh
task init
```

## Build and Run

Wave Terminal uses [Task](https://taskfile.dev) as its build system. All commands below should be run from within the Git repository root directory.

### Development server

Run the following command to build the app and run it via Vite's development server (enables Hot Module Reloading):

```sh
task dev
```

This command will:
- Install dependencies if needed
- Build the Go backend components (wavesrv, wsh)
- Build the embedded documentation
- Start the Electron app with hot reloading

### Standalone

Run the following command to build the app and run it standalone, without the development server:

```sh
task start
```

This is useful for testing the production build without packaging.

### Packaged

Run the following command to generate a production build and package it for distribution:

```sh
task package
```

This will:
- Build all components in production mode
- Package the app using electron-builder
- Place all artifacts in the `make/` directory

**Platform-specific notes:**
- **Linux ARM64**: Run `USE_SYSTEM_FPM=1 task package`
- **All platforms**: The packaging process will automatically detect your platform and architecture

### Individual Components

You can also build individual components:

```sh
# Build only the Go backend components
task build:backend

# Build only the frontend (for development)
npm run build:dev

# Build only the frontend (for production)  
npm run build:prod

# Build the documentation site
task docsite:build:embedded
```

## Debugging

### Frontend logs

You can use the regular Chrome DevTools to debug the frontend application. You can open the DevTools using the keyboard shortcut `Cmd+Option+I` on macOS or `Ctrl+Option+I` on Linux and Windows. Logs will be sent to the Console tab in DevTools.

### Backend logs

Backend logs for the development version of Wave can be found at `~/.waveterm-dev/waveapp.log`. Both the NodeJS backend from Electron and the main Go backend will log here.
