package slim

import "testing"

func testInstalled() map[string]bool {
	return set(
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
		"com.google.android.apps.maps",
		"com.google.android.apps.wellbeing",
		"com.android.vending",
		"com.android.chrome",
		"com.google.android.gms",
		"com.google.android.webview",
		"com.google.android.bluetooth",
		"com.android.systemui",
	)
}

func TestSelectStandard(t *testing.T) {
	selection := Select(testInstalled(), set(), Options{})

	assertSet(t, selection.Disable, set(
		"com.google.android.googlequicksearchbox",
		"com.google.android.apps.photos",
		"com.google.android.apps.maps",
		"com.google.android.apps.wellbeing",
	))
	if len(selection.Kept) != 0 {
		t.Fatalf("Kept = %v, want none", selection.Kept)
	}
	for _, pkg := range selection.Disable {
		if IsProtected(pkg) {
			t.Errorf("selected protected package %s", pkg)
		}
	}
}

func TestSelectAggressive(t *testing.T) {
	selection := Select(testInstalled(), set(), Options{Aggressive: true})
	for _, pkg := range []string{"com.android.vending", "com.android.chrome"} {
		if !contains(selection.Disable, pkg) {
			t.Errorf("aggressive selection is missing %s: %v", pkg, selection.Disable)
		}
	}
}

func TestSelectDoesNotIncludeAggressiveByDefault(t *testing.T) {
	selection := Select(testInstalled(), set(), Options{})
	for _, pkg := range []string{"com.android.vending", "com.android.chrome"} {
		if contains(selection.Disable, pkg) {
			t.Errorf("non-aggressive selection includes %s", pkg)
		}
	}
}

func TestSelectSkipAndKeep(t *testing.T) {
	selection := Select(testInstalled(), set(), Options{Skip: []string{"media"}})
	for _, pkg := range []string{"com.google.android.apps.photos", "com.google.android.apps.maps"} {
		if contains(selection.Disable, pkg) {
			t.Errorf("--skip=media still selected %s", pkg)
		}
	}
	if !contains(selection.Disable, "com.google.android.googlequicksearchbox") {
		t.Error("--skip=media dropped the assistant group too")
	}

	selection = Select(testInstalled(), set(), Options{Keep: []string{"com.google.android.apps.maps"}})
	if contains(selection.Disable, "com.google.android.apps.maps") {
		t.Error("--keep still selected the package")
	}
	if !contains(selection.Kept, "com.google.android.apps.maps") {
		t.Errorf("--keep did not report the package: %v", selection.Kept)
	}
	if !contains(selection.Disable, "com.google.android.apps.photos") {
		t.Error("--keep for one package suppressed another in the same group")
	}
}

func TestSelectSkipsAlreadyDisabled(t *testing.T) {
	disabled := set("com.google.android.apps.photos")
	selection := Select(testInstalled(), disabled, Options{})
	if contains(selection.Disable, "com.google.android.apps.photos") {
		t.Error("selected a package that was already disabled")
	}
}

func TestIsProtected(t *testing.T) {
	for _, pkg := range []string{
		"com.google.android.gms",
		"com.google.android.gms.location",
		"com.google.android.webview",
		"com.google.android.bluetooth",
		"com.android.bluetooth",
		"com.android.systemui",
		"com.android.providers.contacts",
	} {
		if !IsProtected(pkg) {
			t.Errorf("IsProtected(%q) = false, want true", pkg)
		}
	}
	for _, pkg := range []string{
		"com.google.android.apps.photos",
		"com.android.vending",
		"com.google.android.googlequicksearchbox",
		"com.android.bluetoothmidiservice",
	} {
		if IsProtected(pkg) {
			t.Errorf("IsProtected(%q) = true, want false", pkg)
		}
	}
}

func TestValidSkip(t *testing.T) {
	for _, group := range []string{"media", "telephony", "assistant", "bglimit", "animations", "aggressive"} {
		if !ValidSkip(group) {
			t.Errorf("ValidSkip(%q) = false, want true", group)
		}
	}
	for _, group := range []string{"", "gms", "everything"} {
		if ValidSkip(group) {
			t.Errorf("ValidSkip(%q) = true, want false", group)
		}
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
