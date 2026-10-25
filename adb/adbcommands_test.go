package adb

import (
	"reflect"
	"testing"
)

func TestBuildAvdRunArgs(t *testing.T) {
	tests := []struct {
		name         string
		avdName      string
		coldBoot     bool
		emulatorArgs []string
		want         []string
	}{
		{
			name:    "default",
			avdName: "Pixel_7_Pro_API_34",
			want:    []string{"@Pixel_7_Pro_API_34"},
		},
		{
			name:     "cold boot",
			avdName:  "Pixel_7_Pro_API_34",
			coldBoot: true,
			want:     []string{"@Pixel_7_Pro_API_34", "-no-snapshot"},
		},
		{
			name:         "extra emulator args",
			avdName:      "Pixel_7_Pro_API_34",
			emulatorArgs: []string{"-no-window", "-no-audio"},
			want:         []string{"@Pixel_7_Pro_API_34", "-no-window", "-no-audio"},
		},
		{
			name:         "cold boot with extra emulator args",
			avdName:      "Pixel_7_Pro_API_34",
			coldBoot:     true,
			emulatorArgs: []string{"-gpu", "swiftshader_indirect"},
			want:         []string{"@Pixel_7_Pro_API_34", "-no-snapshot", "-gpu", "swiftshader_indirect"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAvdRunArgs(tt.avdName, tt.coldBoot, tt.emulatorArgs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildAvdRunArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
