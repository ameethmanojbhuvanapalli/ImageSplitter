package cli

import "testing"

func TestParseArgs_DefaultGUI(t *testing.T) {
	opts, err := ParseArgs(nil)
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if opts.Run {
		t.Fatalf("expected Run=false")
	}
	if opts.OpenReport {
		t.Fatalf("expected OpenReport=false")
	}
}

func TestParseArgs_SupportsSlashRun(t *testing.T) {
	opts, err := ParseArgs([]string{"/run", "--open-report"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if !opts.Run {
		t.Fatalf("expected Run=true")
	}
	if !opts.OpenReport {
		t.Fatalf("expected OpenReport=true")
	}
}

func TestParseArgs_OpenReportRequiresRun(t *testing.T) {
	_, err := ParseArgs([]string{"--open-report"})
	if err == nil {
		t.Fatalf("expected error when --open-report is used without --run")
	}
}
