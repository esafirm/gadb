package adb

// CommandReturn have output and error of the command
type CommandReturn struct {
	Output []byte
	Error  error
}

// PackageListCommandRetrun have list of packages and error
type PackageListCommandRetrun struct {
	PackageList []string
	Error       error
}
