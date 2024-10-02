package utils

import (
	"fmt"

	adb "github.com/esafirm/gadb/adb"
)

// SelectThirdPartyPackage show package selection
func SelectThirdPartyPackage() string {
	commandReturn := adb.ListPacakgesThirdPartyFormatted()
	return showPackageSelection(commandReturn)
}

func showPackageSelection(commandReturn adb.PackageListCommandRetrun) string {
	if commandReturn.Error != nil {
		fmt.Println("Error getting package list:", commandReturn.Error)
		return ""
	}

	_, selectedPacakge := ShowSelection("Select package", commandReturn.PackageList)
	return selectedPacakge
}
