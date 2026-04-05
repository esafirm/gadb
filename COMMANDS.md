## Global Flags

- `--config [file]`: Specify a custom configuration file (default: `$HOME/.gadb.yaml`)

## Commands

### analyze

Analyze device logs with AI insights. Detect crashes, anomalies, performance issues, and provide actionable insights.

```shell
$ gadb analyze [flags]
```

Flags:
- `-c, --crashes`: Focus on crash logs only (default if no other analysis requested)
- `-p, --performance`: Analyze performance issues
- `-s, --startup`: Analyze app startup performance
- `-r, --recent`: Only analyze recent logs
- `-t, --time [duration]`: Time range for logs (e.g., '5m', '1h')
- `-a, --ai`: Use AI for analysis (requires configuration)
- `--provider [name]`: AI provider: gemini, anthropic, or openai (default: gemini)
- `-v, --verbose`: Show verbose output
- `-k, --package [name]`: Filter logs by package name

### avds 

List available emulators and run them.

```shell
$ gadb avds [emulator_name] [flags]
```

Flags:
- `-w, --wipe`: Wipe AVD data before running
- `-c, --cold`: Run AVD in cold boot state

### battery

Get device battery status.

```shell
$ gadb battery
```

### clear

Trigger clear data to the selected package.

```shell
$ gadb clear [package_name]
```

If `[package_name]` is not provided, it will use the one from the configuration or prompt for selection.

### config

Manage gadb configuration, including AI settings for the `analyze` command.

```shell
$ gadb config [flags]
```

Flags:
- `-a, --ai`: Configure AI settings (provider, API key, model)
- `-s, --show`: Show current configuration

### debug

Set waiting for debugger status. Pretty handy if you want to debug your deeplink or any custom entry point in your app.

```shell
$ gadb debug [package_name] [flags]
``` 

Flags:
- `-p, --persistent`: Set waiting for debug mode until clear is triggered (default: true)
- `-c, --clear`: Clear waiting for debugger status
- `-r, --restart`: Restart the application after setting debug status (default: true)

### focus

Get information about the current focused app, window, and fragment.

```shell
$ gadb focus
```

### install

Install APK to connected device(s). Automatically handles uninstallation if a version downgrade or existing installation is detected.

```shell
$ gadb install [apk_path]
```

### init

Create GADB configuration. Defines your package name and optional AI settings so you don't have to pass them repeatedly.

```shell
$ gadb init
```

### instrumentation

Run instrumentation tests on your device.

```shell
$ gadb instrumentation [package_name] [flags]
```

Flags:
- `-d, --debug`: Enable debug mode
- `-p, --package-selection`: Enable package selection mode

### manifest

Print the `AndroidManifest.xml` from an APK file.

```shell
$ gadb manifest [apk_path]
```

### package

Print the package name from an APK file.

```shell
$ gadb package [apk_path]
```

### restart

Restart the application.

```shell
$ gadb restart [package_name] [flags]
```

Flags:
- `-c, --clear`: Restart application and clear the application data

### start

Start an Android application.

```shell
$ gadb start [package_name]
```

If `[package_name]` is not provided, it will use the one from the configuration.

### store

Open the Play Store page for an application.

```shell 
$ gadb store [package_name]
```

If `[package_name]` is not provided, it will use the one from the configuration.
