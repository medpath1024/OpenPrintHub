//go:build darwin

package printer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDarwinPrintUsesImageFileExtensionForImageJobs(t *testing.T) {
	tempDir := t.TempDir()
	argsPath := filepath.Join(tempDir, "lp-args.txt")
	lpPath := filepath.Join(tempDir, "lp")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(argsPath) + "\n"
	if err := os.WriteFile(lpPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake lp: %v", err)
	}

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	job := &PrintJob{
		PrinterName: "Xprinter XP-420B",
		Type:        JobTypeImage,
		Data:        []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00},
		Settings:    DefaultSettings(),
	}

	if err := (&darwinService{}).Print(job); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	argsBytes, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read fake lp args: %v", err)
	}
	args := strings.Fields(string(argsBytes))
	if len(args) == 0 {
		t.Fatal("fake lp captured no args")
	}

	printPath := args[len(args)-1]
	if filepath.Ext(printPath) != ".png" {
		t.Fatalf("image print temp file extension = %q, want .png; path=%s", filepath.Ext(printPath), printPath)
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
