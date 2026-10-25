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
- `-n, --no-window`: Run AVD without an emulator window (headless)
- `-e, --emulator-arg [arg]`: Extra emulator argument, repeatable

```shell
# Run headless
$ gadb avds Pixel_7_Pro_API_34 --no-window

# Pass any other emulator option through
$ gadb avds Pixel_7_Pro_API_34 -e -no-audio -e -gpu -e swiftshader_indirect
```

### battery

Get device battery status.

```shell
$ gadb battery
```

### cleanup

Free disk space on a running emulator. Only devices verified as emulators are cleaned (checked via `ro.kernel.qemu`, `ro.boot.qemu`, `ro.hardware`, and `ro.build.characteristics`) — physical devices are never touched.

```shell
$ gadb cleanup [flags]
```

Cleaned by default:
- Perfetto traces: `/data/misc/perfetto-traces`
- Crash tombstones: `/data/tombstones`
- System debris: `package_cache`, ANR traces, dropbox, textclassifier models, resource cache (all regenerated on demand)
- Temp files: `/data/local/tmp` (keeps `lldb-server`, `.studio`, `avdslim_state.json`, and other tooling)
- App caches: `pm trim-caches`

Flags:
- `-s, --serial [serial]`: Target a specific emulator (default: auto-detect, or prompt when several are running)
- `-n, --dry-run`: Show categories and sizes without deleting anything
- `-y, --yes`: Skip the confirmation prompt
- `--sdcard`: Also empty `/sdcard/Download`, `/sdcard/Pictures` and `/sdcard/DCIM`
- `--apps`: Interactively pick third-party apps to uninstall (your configured package, `devicelab`, `androidx.test.*` and `com.android.cli.*` are always kept)
- `--keep [prefix]`: Never uninstall packages matching this prefix (repeatable)

Root is used when available (`su 0` first, otherwise `adb root`, restored afterwards). Without root, Perfetto traces, tombstones and system debris are skipped.

Examples:
- `gadb cleanup` — clean the only connected emulator
- `gadb cleanup -n` — preview what would be freed
- `gadb cleanup --sdcard --apps` — also clear shared storage, then pick apps to uninstall
- `gadb cleanup -s emulator-5554 -y` — non-interactive cleanup of a specific emulator

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

### log / logs

Interactive terminal UI for streaming, searching, and analyzing device logcat in real time (inspired by [byedroid](https://github.com/cesarferreira/byedroid)).

Aliases: `logcat`

```shell
$ gadb log [package] [filter] [flags]
$ gadb logs [package] [filter] [flags]
$ gadb logcat
```

Parameters:
- `[package]`: Optional application ID / package name to target (e.g. `com.example.app`). If omitted, uses the configured package or defaults to all packages.
- `[filter]`: Optional initial search / include filter query (e.g. `"MainActivity"` or `Retrofit error`).

Flags:
- `-k, --package [name]`: Filter logs by application ID / package name (defaults to project config if available)
- `-s, --serial [serial]`: Target specific device serial
- `-l, --level [level]`: Minimum log level (V, D, I, W, E)
- `-a, --all`: Start showing all logs (no package filter)
- `-f, --filter [query]`: Initial search / include filter
- `-x, --exclude [query]`: Initial exclude filter
- `-m, --mouse`: Enable mouse capture (wheel scroll). Off by default so logs stay selectable/copyable

Examples:
- `gadb log` — start logcat TUI for configured package or all packages
- `gadb logs` — same as `gadb log`
- `gadb log com.example.app` — target specific package immediately
- `gadb logs com.example.app "Network"` — target package with initial filter query
- `gadb log -a "fatal"` — view all device logs with initial search query

Keybindings:
| Key | Action |
|-----|--------|
| `Space` | Pause / resume log streaming |
| `↑` `↓` `j` `k` / mouse wheel (when mouse ON) | Scroll logcat viewport |
| `PageUp` `PageDown` | Scroll 20 lines |
| `End` `G` | Jump to bottom (follow tail) |
| `m` | Toggle mouse capture (`OFF` = drag to select & copy, `ON` = wheel scroll) |
| `f` | Include filter (live search tag, message, level) |
| `x` | Exclude filter (live exclude noise) |
| `L` | Log level picker overlay (All, D, I, W, E) |
| `a` | Toggle all logs vs package-filtered logs |
| `p` | Target package input & picker modal |
| `c` | Clear log buffer (and device logcat buffer) |
| `w` | Export captured logs to `gadb-log-<timestamp>.log` |
| `y` | Crash & ANR detail popup (`c` copy stack · `a` copy agent prompt · `w` export · `s` search web) |
| `?` or `h` | Toggle help modal |
| `q` / `Ctrl+C` | Quit |

Copying logs: mouse capture is OFF by default, so drag-select text with the mouse and copy with `Cmd/C` or `Ctrl+Shift+C`. Press `m` (or pass `--mouse`) only when you want wheel-scroll at the cost of native selection. `w` exports all visible logs to a file as a selection-free alternative.

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

### settings

Toggle common device settings with a filterable TUI (type to filter, Tab toggles
instantly, Enter opens Toggle / Turn ON / Turn OFF). State is shown live with colors
(green ON, red OFF, yellow unset/error) in each row.
Covers e.g. Don't keep activities, Stay awake (always on when connected to power), Show taps, animation scales, auto-rotate, airplane mode.

Alias: `setting`

```shell
$ gadb settings
$ gadb setting
$ gadb settings --list
$ gadb settings [key-or-name] [on|off|toggle]
```

Examples:
- `gadb settings` — open the filterable list, Tab toggles instantly or Enter opens the action menu (repeats until you exit)
- `gadb settings --list` — print current values with colors, without the TUI
- `gadb settings always_finish_activities` — toggle Don't keep activities directly
- `gadb settings show_touches on` — enable Show taps directly
- `gadb settings airplane_mode_on toggle` — toggle Airplane mode directly
- `gadb settings always on when connected to power on` — keep screen always on while plugged in
- `gadb settings "stay awake" toggle` — toggle stay awake while charging

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
