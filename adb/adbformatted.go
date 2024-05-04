package adb

// List down the 3rd party pacakges in a formatted way
func ListPacakgesThirdPartyFormatted() PackageListCommandRetrun {
	commandReturn := ListPackageThirdParty()
	if commandReturn.Error != nil {
		return PackageListCommandRetrun{Error: commandReturn.Error}
	}
	return PackageListCommandRetrun{PackageList: splitPackageList(
		string(commandReturn.Output),
		"package:",
	)}
}
