package main

import (
	"fmt"
	"strings"
)

func main() {
	text := `output adb: failed to install /Users/esafirm/Downloads/sibalec_debug_1029.apk: Failure [INSTALL_FAILED_ALREADY_EXISTS: Attempt to re-install          sibalecx.android without first uninstalling.]`
	var isAlreadyExistProblem = strings.Contains(text, "ALREADY_EXISTS")
	var index = strings.Index(text, "re-install") + len("re-install")
	var withoutIndex = strings.Index(text, "without")
	var packageName = strings.TrimSpace(text[index:withoutIndex])

	fmt.Println("Package name:", packageName)
	fmt.Println("Already exist:", isAlreadyExistProblem)
	fmt.Println("Index: ", index)
}
