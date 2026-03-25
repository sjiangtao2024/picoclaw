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
	installRoot := filepath.Join(tmpDir, "root", "picoclaw")
	binDir := filepath.Join(installRoot, "bin")
	configDir := filepath.Join(installRoot, "config")
	logDir := filepath.Join(installRoot, "logs")
	servicePath := filepath.Join(tmpDir, "systemd", "picoclaw-web.service")
	commandBinDir := filepath.Join(tmpDir, "usr-local-bin")
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
		"SERVICE_PATH="+servicePath,
		"COMMAND_BIN_DIR="+commandBinDir,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install script failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(binDir, "picoclaw-web")); err != nil {
		t.Fatalf("launcher binary missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "picoclaw")); err != nil {
		t.Fatalf("gateway binary missing: %v", err)
	}
	if _, err := os.Stat(servicePath); err != nil {
		t.Fatalf("service file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "config.json")); err != nil {
		t.Fatalf("config.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "launcher-config.json")); err != nil {
		t.Fatalf("launcher-config.json missing: %v", err)
	}
	if _, err := os.Stat(logDir); err != nil {
		t.Fatalf("logs dir missing: %v", err)
	}
	for _, wrapperName := range []string{"picoclaw", "picoclaw-web"} {
		wrapperPath := filepath.Join(commandBinDir, wrapperName)
		wrapperData, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", wrapperPath, err)
		}
		wrapperText := string(wrapperData)
		for _, want := range []string{
			installRoot,
			filepath.Join(configDir, "config.json"),
		} {
			if !strings.Contains(wrapperText, want) {
				t.Fatalf("wrapper %s missing %q:\n%s", wrapperName, want, wrapperText)
			}
		}
	}

	serviceData, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("ReadFile(servicePath): %v", err)
	}
	serviceText := string(serviceData)
	for _, want := range []string{
		"WorkingDirectory="+installRoot,
		"ExecStart=" + filepath.Join(installRoot, "bin", "picoclaw-web") + " --no-browser " + filepath.Join(installRoot, "config", "config.json"),
		"Environment=PICOCLAW_HOME=" + installRoot,
	} {
		if !strings.Contains(serviceText, want) {
			t.Fatalf("service file missing %q:\n%s", want, serviceText)
		}
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
	installRoot := filepath.Join(tmpDir, "root", "picoclaw")
	binDir := filepath.Join(installRoot, "bin")
	backupDir := filepath.Join(installRoot, "backups")
	commandBinDir := filepath.Join(tmpDir, "usr-local-bin")
	fakeBinDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fakeBinDir): %v", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(binDir): %v", err)
	}

	systemctlLog := filepath.Join(tmpDir, "systemctl.log")
	systemctlStub := filepath.Join(fakeBinDir, "systemctl")
	if err := os.WriteFile(systemctlStub, []byte("#!/bin/sh\necho \"$@\" >> "+systemctlLog+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(systemctlStub): %v", err)
	}

	if err := os.WriteFile(filepath.Join(binDir, "picoclaw-web"), []byte("old-web"), 0o755); err != nil {
		t.Fatalf("WriteFile(old web): %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "picoclaw"), []byte("old-gateway"), 0o755); err != nil {
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
		"COMMAND_BIN_DIR="+commandBinDir,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("upgrade script failed: %v\n%s", err, out)
	}

	webData, err := os.ReadFile(filepath.Join(binDir, "picoclaw-web"))
	if err != nil {
		t.Fatalf("ReadFile(installed web): %v", err)
	}
	if string(webData) != "new-web" {
		t.Fatalf("installed web binary = %q, want %q", string(webData), "new-web")
	}

	gatewayData, err := os.ReadFile(filepath.Join(binDir, "picoclaw"))
	if err != nil {
		t.Fatalf("ReadFile(installed gateway): %v", err)
	}
	if string(gatewayData) != "new-gateway" {
		t.Fatalf("installed gateway binary = %q, want %q", string(gatewayData), "new-gateway")
	}
	for _, wrapperName := range []string{"picoclaw", "picoclaw-web"} {
		if _, err := os.Stat(filepath.Join(commandBinDir, wrapperName)); err != nil {
			t.Fatalf("wrapper %s missing: %v", wrapperName, err)
		}
	}

	backups, err := filepath.Glob(filepath.Join(backupDir, "*"))
	if err != nil {
		t.Fatalf("Glob(backups): %v", err)
	}
	if len(backups) < 2 {
		t.Fatalf("backup files = %d, want at least 2", len(backups))
	}
}
