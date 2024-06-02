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

// List down AVD(s) and ignore other data not related to it such as
// crash report path info
func AvdListFormatted() AvdListCommandReturn {
	commandReturn := AvdList()
	if commandReturn.Error != nil {
		return AvdListCommandReturn{Error: commandReturn.Error}
	}
	return AvdListCommandReturn{AvdList: splitAvdList(string(commandReturn.Output))}
}
