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

### slim

Keep Android emulators light, fast and reproducible. Run `gadb slim` with no
subcommand for an interactive menu; the subcommands and flags are for scripting.

`check`, `create` and `tune` never change the guest. `on` and `snapshot --live`
change a running guest, `off` reverses them, and `snapshot` bakes a clean state.

```shell
$ gadb slim check [avd]
$ gadb slim create <name> [flags]
$ gadb slim tune [avd] [flags]
$ gadb slim on [serial|avd] [flags]
$ gadb slim off [serial|avd]
$ gadb slim status [serial|avd]
$ gadb slim start [avd] [flags]
$ gadb slim snapshot [avd] [flags]
$ gadb slim unbake [avd] [flags]
```

`check` is read-only. It inspects the AVD's system image and `config.ini`, and when
the emulator is running it reads the real guest RAM, renderer and low-RAM state.

`create` makes a new AVD from the newest complete 4 KB `google_apis` image for this
host, then tunes it. It refuses to overwrite an existing name and never downloads
images. When nothing suitable is installed it prints the exact `sdkmanager` command.
`avdmanager` must be available and needs Java 17+.

`tune` writes `config.ini` for a stopped AVD:

- `hw.ramSize` (default 2048 MB, `--ram=1536` for maximum savings)
- `hw.gpu.mode` (host on macOS/Windows, auto on Linux desktop, swiftshader_indirect on headless Linux)
- audio and camera emulation off
- `hardware-qemu.ini` removed so the emulator regenerates it

The original values are recorded in `gadb-slim.json` next to `config.ini`, plus an
untouched `config.ini.gadb.bak`. `--restore` puts back only the values gadb changed,
so edits you made after tuning are left alone.

`on` prunes a running, verified emulator. It disables the curated package groups,
sets `background_process_limit=4` and merges `max_cached_processes=4` into
`activity_manager_constants`, then trims cached processes. The restore record is
saved before anything changes, so a crash halfway through can still be undone.

`off` re-enables the packages and restores the settings recorded for that emulator,
then restores `config.ini`. If the emulator is stopped, guest changes stay in
`gadb-slim.json` and are restored the next time `off` runs with it online.

`status` shows the recorded package and setting counts, config drift, and whether
the running guest still matches the manifest.

`start` tunes a stopped AVD, boots it with `-lowram`, the host GPU, no audio and no
boot animation, waits for boot and runs `on`. It restores `gadb_clean` when the
snapshot is valid and current, otherwise it cold boots. The emulator keeps running
after gadb exits. `--cold` ignores the snapshot; `--no-slim` skips guest pruning.
If the AVD is already tuned, `start` keeps its RAM and GPU unless you pass a
tuning flag on that run.

`snapshot` (alias `bake`) cold boots a stopped AVD, applies `on`, saves
`gadb_clean` and shuts the emulator down. If the AVD is running it stops it first,
prompting unless `--yes`. `--live` snapshots the running guest instead, including
installed apps and data, and cannot be combined with `--headless`, `--ram`, `--gpu`,
`--keep-audio` or `--keep-camera`. The snapshot is tied to the current system image
and `config.ini`, and `status`/`check` report it as stale when either changes. A
cold snapshot keeps an existing tune unless you pass a tuning flag.

`unbake` deletes `gadb_clean` and clears its record. It stops a running AVD first,
prompting unless `--yes`.

Create flags:
- `--api [level]`: API level to use (default: newest installed)
- `--device [id]`: `avdmanager` device profile (default `pixel`)
- `--ram [mb]`: Guest RAM in MB (default 2048)
- `--gpu [mode]`: Emulator GPU mode (default host on macOS/Windows)
- `--keep-audio`: Leave audio emulation on
- `--keep-camera`: Leave camera emulation on

Tune flags:
- `--ram [mb]`: Guest RAM in MB (default 2048)
- `--gpu [mode]`: Emulator GPU mode (default host on macOS/Windows)
- `--keep-audio`: Leave audio emulation on
- `--keep-camera`: Leave camera emulation on
- `--restore`: Restore the values recorded by the last tune

On flags:
- `--aggressive`: Also disable Play Store, Chrome, partner setup and setup wizard
- `--keep [package]`: Never disable this package (repeatable). The configured app package is kept automatically
- `--skip [group]`: Leave a group alone (repeatable): assistant, peripherals, privacy, wellbeing, media, telephony, accessibility, printing, telemetry, aggressive, animations, bglimit
- `--no-anim`: Set the animation scales to 0 (default: leave them alone)

Start flags:
- `--ram [mb]`, `--gpu [mode]`: guest RAM and GPU mode
- `--headless`: no emulator window
- `--cold`: ignore the golden snapshot and cold boot
- `--no-lowram`: do not pass `-lowram`
- `--no-slim`: boot without running `on`
- `--keep-audio`, `--keep-camera`: leave those emulations on
- `--aggressive`, `--keep`, `--skip`, `--no-anim`: same as `on`

Snapshot flags:
- `--live`: snapshot the running guest instead of a cold bake
- `--headless`: bake without an emulator window
- `--yes`: stop a running AVD without asking
- `--ram [mb]`, `--gpu [mode]`: guest RAM and GPU mode
- `--keep-audio`, `--keep-camera`: leave those emulations on
- `--aggressive`, `--keep`, `--skip`, `--no-anim`: same as `on`

Unbake flags:
- `--yes`: stop a running AVD without asking

Core packages are never disabled: GMS and GSF, WebView, the keyboard, launcher,
SystemUI, permission controllers, package installers, providers and Bluetooth.

Examples:
- `gadb slim check` — audit every installed AVD, or one in detail when only one exists
- `gadb slim check Pixel_7_Pro_API_34` — detailed report for one AVD
- `gadb slim create Pixel_Slim` — newest complete 4 KB image, then tune
- `gadb slim create Pixel_Slim --api=35 --device=pixel_6 --ram=1536` — pick the API and profile
- `gadb slim tune Pixel_7_Pro_API_34 --ram=1536` — maximum memory savings
- `gadb slim on` — prune the only running emulator
- `gadb slim on Pixel_7_Pro_API_34` — prune a specific AVD
- `gadb slim on emulator-5554 --aggressive` — include Play Store and Chrome
- `gadb slim on --keep=com.google.android.apps.maps --skip=telephony` — keep Maps and the phone apps
- `gadb slim status` — check what is applied and whether the guest drifted
- `gadb slim off` — restore the running emulator and `config.ini`
- `gadb slim off Pixel_7_Pro_API_34` — restore an AVD even when it is stopped
- `gadb slim start Pixel_Slim` — tune, boot and prune in one command
- `gadb slim start Pixel_Slim --headless --ram=1536` — headless with maximum savings
- `gadb slim start Pixel_Slim --cold --skip=telephony` — ignore the snapshot and keep the phone apps
- `gadb slim snapshot Pixel_Slim` — cold bake a clean `gadb_clean` snapshot
- `gadb slim snapshot Pixel_Slim --live` — capture the running guest as the snapshot
- `gadb slim unbake Pixel_Slim` — delete the golden snapshot

A Play Store or 16 KB page-size image keeps its own RAM floor. 16 KB images reserve
4096 MB regardless of `hw.ramSize`, so `check` will suggest `gadb slim create` for a
replacement AVD built from a 4 KB `google_apis` image.

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
