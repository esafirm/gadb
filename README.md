## GADB (Go ADB)

> Work in progress

ADB wrapper with enhanced and more features! 

## Usage 

Do anything you want to do with `adb` 

For example you can get the list of connected android devices by using `gadb devices`

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

Install apk to connected devices. Please note this command will automatically uninstall and install if `adb install` return `ALREADY_EXISTS` error 

```shell
$ gadb install [apk_path]
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

## License

Apache 2 @ Esa Firman