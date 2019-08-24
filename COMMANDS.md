## Commands

### avds 

List available emulators 

```
$ gadb avds
```

Run emulator 

```
$ gadb avds [emulator_name]
```

### debug

Set waiting for debugger status, pretty handy if you want to debug your deeplink or any custom entry point in your app

```
$ gadb debug [package_name] [flags]
``` 

Clear waiting for debugger status 

```
$ gadb debug --clear 
```

### install

Install apk to connected devices.

> This command will automatically uninstall and install if `adb install` return `ALREADY_EXISTS` error 

```shell
$ gadb install [apk_path]
```

### init

Create GADB configuration. With this configuration you can define your package name and another info so you don't have to pass it again to the comamnds.
For example, you can use `gadb start` without specifying the `[package name]`. It will be fetched from your configuration.

```shell
$ gadb init
```

### restart

Restart application

```shell
$ gadb restart [package_name]
```

Restart application and clear the application data

```shell
$ gadb restart [package_name] --clear
```

### store

Open PlayStore page

```shell 
$ gadb store [package_name]
```