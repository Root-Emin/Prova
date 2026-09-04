package model

import "testing"

// Eski normalizasyon yalnızca kanonik adları tanıyordu; Electron'un
// process.platform değerleri ("win32", "darwin") bunlara girmiyordu ve gerçek
// masaüstü istemcisinden gelen her cihaz "unknown" olarak kaydediliyordu.
func TestNormalizePlatform_RecognisesElectronAndBrowserValues(t *testing.T) {
	cases := map[string]DevicePlatform{
		// Electron / Node process.platform
		"win32":  DevicePlatformWindows,
		"darwin": DevicePlatformMacOS,
		"linux":  DevicePlatformLinux,
		// navigator.platform
		"Win32":        DevicePlatformWindows,
		"MacIntel":     DevicePlatformMacOS,
		"Linux x86_64": DevicePlatformLinux,
		// kanonik adlar
		"windows": DevicePlatformWindows,
		"macos":   DevicePlatformMacOS,
		"web":     DevicePlatformWeb,
		// bilinmeyen
		"":       DevicePlatformUnknown,
		"plan9":  DevicePlatformUnknown,
		"  ios ": DevicePlatformUnknown,
	}
	for input, want := range cases {
		if got := NormalizePlatform(input); got != want {
			t.Errorf("NormalizePlatform(%q) = %q, beklenen %q", input, got, want)
		}
	}
}

// Üç masaüstü platformunun tamamı doğru eşleşmeli; Faz 7'nin kontrol maddesi
// tam olarak budur.
func TestNormalizePlatform_CoversAllThreeDesktopPlatforms(t *testing.T) {
	seen := map[DevicePlatform]bool{}
	for _, input := range []string{"win32", "darwin", "linux"} {
		seen[NormalizePlatform(input)] = true
	}
	for _, want := range []DevicePlatform{DevicePlatformWindows, DevicePlatformMacOS, DevicePlatformLinux} {
		if !seen[want] {
			t.Errorf("%q platformu normalize edilemedi", want)
		}
	}
}

// Parmak izi düz metin saklanmamalı; hash deterministik ve boş girdide boş
// olmalı, yoksa anahtar çifti olmayan cihazlar aynı hash'e çakışır.
func TestHashFingerprint(t *testing.T) {
	first := HashFingerprint("abc-123-device")
	if first == "" || first == "abc-123-device" {
		t.Fatalf("parmak izi hash'lenmeli, %q geldi", first)
	}
	if second := HashFingerprint("abc-123-device"); second != first {
		t.Fatal("hash deterministik olmalı")
	}
	if HashFingerprint("") != "" {
		t.Fatal("boş parmak izi boş hash üretmeli")
	}
}
