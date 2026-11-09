package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProbeDoesNotAcceptMereAvailability(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required by the production probe")
	}
	script, err := filepath.Abs("../../.github/scripts/probe-readings.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, json string
		success    bool
	}{
		{"legacy", `{"ref":"Lc 17,1-6"}`, false},
		{"unverified", `{"ref":"Lc 17,1-6","verified":false,"source_url":"https://example.test"}`, false},
		{"verified", `{"ref":"Ga 2,13-22","verified":true,"source_url":"https://www.vaticannews.va/vi/loi-chua-hang-ngay/2026/11/09.html"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cli := filepath.Join(dir, "cli")
			if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf '%s\\n' '"+tc.json+"'\n"), 0755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", script)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "CLI="+cli, "LOOKAHEAD_DAYS=0")
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.success {
				t.Fatalf("success=%v, err=%v output=%s", tc.success, err, out)
			}
			for _, name := range []string{"probe-years.txt", "probe-from.txt", "probe-to.txt"} {
				if data, err := os.ReadFile(filepath.Join(dir, "build", name)); err != nil || len(data) == 0 {
					t.Fatalf("missing bounded crawl output %s: %v", name, err)
				}
			}
		})
	}
}
