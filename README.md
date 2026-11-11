## GADB (Go ADB)

ADB wrapper with enhanced and more features!

## Install

Using [Homebrew](https://brew.sh/)

```
brew install esafirm/tap/gadb
```

Or download the binareis in [release page](https://github.com/esafirm/gadb/releases)

Create alias!

```
alias adb=gadb
```

## Usage

Do anything you want to do with `adb`

For example you can get the list of connected android devices by using `gadb devices`

GADB uses the best backend for each operation. Install [Android CLI](https://developer.android.com/cli)
to enable delta APK installation, build-and-run, JSON layouts, selector-based input,
annotated screenshots, profiling, and artifact analysis. Existing ADB commands,
log tools, and emulator management continue to work without it.

`gadb install` prefers Android CLI and falls back to ADB only when Android CLI is
not installed. A failed Android CLI command is not silently retried through ADB.

## Commands

For detailed command, please refer to [Command Page](https://github.com/esafirm/gadb/blob/master/COMMANDS.md)

## License

Apache 2 @ Esa Firman
