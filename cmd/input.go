package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var inputDevice string

var inputCmd = &cobra.Command{Use: "input", Short: "Interact with UI elements or send coordinate input"}
var semanticInputCmd = &cobra.Command{
	Use:                "semantic <key|scroll|swipe|tap|text> [arguments...]",
	Short:              "Perform an accessibility-selector based input operation",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("specify an android input operation (key, scroll, swipe, tap, or text)")
		}
		device := inputDevice
		operation := 0
		for operation < len(args) && (args[operation] == "--device" || args[operation] == "-s" || strings.HasPrefix(args[operation], "--device=")) {
			arg := args[operation]
			if strings.HasPrefix(arg, "--device=") {
				device = strings.TrimPrefix(arg, "--device=")
				if device == "" {
					return fmt.Errorf("--device requires a device serial")
				}
			} else {
				if operation+1 == len(args) || args[operation+1] == "--" {
					return fmt.Errorf("%s requires a device serial", arg)
				}
				device = args[operation+1]
				operation++
			}
			operation++
		}
		androidArgs := []string{"input"}
		if device != "" {
			androidArgs = append(androidArgs, "--device", device)
		}
		androidArgs = append(androidArgs, args[operation:]...)
		return runToolCommand(cmd, "android", androidArgs...)
	},
}
var coordinateCmd = &cobra.Command{Use: "coord", Short: "Send coordinate-based input using adb"}

func coordinateOperation(operation string, count int) *cobra.Command {
	usage := map[string]string{"tap": "x y", "swipe": "x1 y1 x2 y2 [duration-ms]", "text": "text", "keyevent": "keycode"}[operation]
	return &cobra.Command{Use: operation + " <" + usage + ">", Short: "Send coordinate-based " + operation + " input", Args: func(_ *cobra.Command, args []string) error {
		if len(args) < count || (operation != "swipe" && len(args) != count) || (operation == "swipe" && len(args) > 5) {
			return fmt.Errorf("invalid argument count")
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		if operation == "tap" || operation == "swipe" {
			coords := 2
			if operation == "swipe" {
				coords = 4
			}
			for i := 0; i < coords; i++ {
				if _, err := strconv.Atoi(args[i]); err != nil {
					return fmt.Errorf("coordinate %q must be an integer", args[i])
				}
			}
			if operation == "swipe" && len(args) == 5 {
				if _, err := strconv.Atoi(args[4]); err != nil {
					return fmt.Errorf("swipe duration %q must be integer milliseconds", args[4])
				}
			}
		}
		if operation == "keyevent" {
			if _, err := strconv.Atoi(args[0]); err != nil {
				return fmt.Errorf("keycode %q must be an integer", args[0])
			}
		}
		adbArgs := []string{}
		if inputDevice != "" {
			adbArgs = append(adbArgs, "-s", inputDevice)
		}
		adbArgs = append(adbArgs, "shell", "input", operation)
		if operation == "text" {
			adbArgs = append(adbArgs, shellQuote(strings.ReplaceAll(args[0], " ", "%s")))
		} else {
			for _, arg := range args {
				adbArgs = append(adbArgs, shellQuote(arg))
			}
		}
		return runToolCommand(cmd, "adb", adbArgs...)
	}}
}

var shellMeta = regexp.MustCompile(`[^A-Za-z0-9_.,%+:/=-]`)

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !shellMeta.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func init() {
	inputCmd.PersistentFlags().StringVarP(&inputDevice, "device", "s", "", "Device serial number")
	inputCmd.AddCommand(semanticInputCmd)
	coordinateCmd.AddCommand(coordinateOperation("tap", 2), coordinateOperation("swipe", 4), coordinateOperation("text", 1), coordinateOperation("keyevent", 1))
	inputCmd.AddCommand(coordinateCmd)
	rootCmd.AddCommand(inputCmd)
}
