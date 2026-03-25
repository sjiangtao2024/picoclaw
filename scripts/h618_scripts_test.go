package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallH618WebInstallsLauncherAndGatewayBinaries(t *testing.T) {
	repoRoot := "."
	scriptPath := "./install-h618-web.sh"

	tmpDir := t.TempDir()
	installRoot := filepath.Join(tmpDir, "opt", "picoclaw", "current")
	dataRoot := filepath.Join(tmpDir, "data", "picoclaw")
	servicePath := filepath.Join(tmpDir, "systemd", "picoclaw-web.service")
	fakeBinDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fakeBinDir): %v", err)
	}

	systemctlLog := filepath.Join(tmpDir, "systemctl.log")
	systemctlStub := filepath.Join(fakeBinDir, "systemctl")
	if err := os.WriteFile(systemctlStub, []byte("#!/bin/sh\necho \"$@\" >> "+systemctlLog+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(systemctlStub): %v", err)
	}

	webBinary := filepath.Join(tmpDir, "picoclaw-web-linux-arm64")
	if err := os.WriteFile(webBinary, []byte("web"), 0o755); err != nil {
		t.Fatalf("WriteFile(webBinary): %v", err)
	}

	gatewayBinary := filepath.Join(tmpDir, "picoclaw")
	if err := os.WriteFile(gatewayBinary, []byte("gateway"), 0o755); err != nil {
		t.Fatalf("WriteFile(gatewayBinary): %v", err)
	}

	cmd := exec.Command("bash", scriptPath, webBinary, gatewayBinary)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"PATH="+fakeBinDir+":"+os.Getenv("PATH"),
		"INSTALL_ROOT="+installRoot,
		"DATA_ROOT="+dataRoot,
		"SERVICE_PATH="+servicePath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install script failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(installRoot, "picoclaw-web-linux-arm64")); err != nil {
		t.Fatalf("launcher binary missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installRoot, "picoclaw")); err != nil {
		t.Fatalf("gateway binary missing: %v", err)
	}
	if _, err := os.Stat(servicePath); err != nil {
		t.Fatalf("service file missing: %v", err)
	}

	logData, err := os.ReadFile(systemctlLog)
	if err != nil {
		t.Fatalf("ReadFile(systemctlLog): %v", err)
	}
	logText := string(logData)
	for _, want := range []string{"daemon-reload", "enable --now picoclaw-web", "status picoclaw-web --no-pager"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("systemctl log missing %q:\n%s", want, logText)
		}
	}
}

func TestUpgradeH618WebInstallsUpdatedGatewayBinary(t *testing.T) {
	repoRoot := "."
	scriptPath := "./upgrade-h618-web.sh"

	tmpDir := t.TempDir()
	installRoot := filepath.Join(tmpDir, "opt", "picoclaw", "current")
	backupDir := filepath.Join(tmpDir, "opt", "picoclaw", "backups")
	fakeBinDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fakeBinDir): %v", err)
	}
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(installRoot): %v", err)
	}

	systemctlLog := filepath.Join(tmpDir, "systemctl.log")
	systemctlStub := filepath.Join(fakeBinDir, "systemctl")
	if err := os.WriteFile(systemctlStub, []byte("#!/bin/sh\necho \"$@\" >> "+systemctlLog+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(systemctlStub): %v", err)
	}

	if err := os.WriteFile(filepath.Join(installRoot, "picoclaw-web-linux-arm64"), []byte("old-web"), 0o755); err != nil {
		t.Fatalf("WriteFile(old web): %v", err)
	}
	if err := os.WriteFile(filepath.Join(installRoot, "picoclaw"), []byte("old-gateway"), 0o755); err != nil {
		t.Fatalf("WriteFile(old gateway): %v", err)
	}

	newWebBinary := filepath.Join(tmpDir, "new-picoclaw-web-linux-arm64")
	if err := os.WriteFile(newWebBinary, []byte("new-web"), 0o755); err != nil {
		t.Fatalf("WriteFile(newWebBinary): %v", err)
	}
	newGatewayBinary := filepath.Join(tmpDir, "new-picoclaw")
	if err := os.WriteFile(newGatewayBinary, []byte("new-gateway"), 0o755); err != nil {
		t.Fatalf("WriteFile(newGatewayBinary): %v", err)
	}

	cmd := exec.Command("bash", scriptPath, newWebBinary, newGatewayBinary)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"PATH="+fakeBinDir+":"+os.Getenv("PATH"),
		"INSTALL_ROOT="+installRoot,
		"BACKUP_DIR="+backupDir,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("upgrade script failed: %v\n%s", err, out)
	}

	webData, err := os.ReadFile(filepath.Join(installRoot, "picoclaw-web-linux-arm64"))
	if err != nil {
		t.Fatalf("ReadFile(installed web): %v", err)
	}
	if string(webData) != "new-web" {
		t.Fatalf("installed web binary = %q, want %q", string(webData), "new-web")
	}

	gatewayData, err := os.ReadFile(filepath.Join(installRoot, "picoclaw"))
	if err != nil {
		t.Fatalf("ReadFile(installed gateway): %v", err)
	}
	if string(gatewayData) != "new-gateway" {
		t.Fatalf("installed gateway binary = %q, want %q", string(gatewayData), "new-gateway")
	}

	backups, err := filepath.Glob(filepath.Join(backupDir, "*"))
	if err != nil {
		t.Fatalf("Glob(backups): %v", err)
	}
	if len(backups) < 2 {
		t.Fatalf("backup files = %d, want at least 2", len(backups))
	}
}
