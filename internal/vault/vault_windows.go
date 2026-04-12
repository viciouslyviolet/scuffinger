//go:build windows

package vault

import (
	"fmt"
	"os/exec"
	"strings"
)

// windowsStore uses PowerShell and the Windows Credential Manager.
type windowsStore struct{}

func newPlatformStore() Store { return &windowsStore{} }

func (s *windowsStore) Set(key, value string) error {
	// Remove any existing entry first.
	_ = s.Delete(key)

	target := serviceName + ":" + key

	// Use cmdkey.exe which is available on all Windows versions.
	// /generic: target  /user: key  /pass: value
	script := fmt.Sprintf(
		`$cred = New-Object System.Management.Automation.PSCredential('%s', (ConvertTo-SecureString '%s' -AsPlainText -Force)); `+
			`New-StoredCredential -Target '%s' -UserName '%s' -SecurePassword $cred.Password -Type Generic -Persist LocalMachine 2>$null; `+
			`if ($LASTEXITCODE -ne 0) { `+
			`  [void][Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]; `+
			`  $v = New-Object Windows.Security.Credentials.PasswordVault; `+
			`  $c = New-Object Windows.Security.Credentials.PasswordCredential('%s','%s','%s'); `+
			`  $v.Add($c) `+
			`}`,
		key, value, target, key, serviceName, key, value,
	)

	// Fallback: simpler approach using cmdkey
	cmd := exec.Command("cmdkey", "/generic:"+target, "/user:"+key, "/pass:"+value)
	if out, err := cmd.CombinedOutput(); err != nil {
		// If cmdkey fails, try PowerShell PasswordVault
		psCmd := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf(
				`[void][Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]; `+
					`$v = New-Object Windows.Security.Credentials.PasswordVault; `+
					`try { $old = $v.Retrieve('%s','%s'); $v.Remove($old) } catch {}; `+
					`$c = New-Object Windows.Security.Credentials.PasswordCredential('%s','%s','%s'); `+
					`$v.Add($c)`,
				serviceName, key, serviceName, key, value,
			),
		)
		if psOut, psErr := psCmd.CombinedOutput(); psErr != nil {
			return fmt.Errorf("credential set %q: cmdkey: %s — powershell: %s", key, strings.TrimSpace(string(out)), strings.TrimSpace(string(psOut)))
		}
		_ = script // suppress unused
	}
	return nil
}

func (s *windowsStore) Get(key string) (string, error) {
	target := serviceName + ":" + key

	// Try cmdkey first
	cmd := exec.Command("cmdkey", "/list:"+target)
	if out, err := cmd.Output(); err == nil && strings.Contains(string(out), target) {
		// cmdkey doesn't expose the password; use PowerShell PasswordVault
	}

	psCmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(
			`[void][Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]; `+
				`$v = New-Object Windows.Security.Credentials.PasswordVault; `+
				`try { $c = $v.Retrieve('%s','%s'); $c.RetrievePassword(); Write-Output $c.Password } `+
				`catch { exit 1 }`,
			serviceName, key,
		),
	)
	out, err := psCmd.Output()
	if err != nil {
		return "", ErrNotFound
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *windowsStore) Delete(key string) error {
	target := serviceName + ":" + key

	// Try cmdkey
	cmd := exec.Command("cmdkey", "/delete:"+target)
	_ = cmd.Run()

	// Also try PasswordVault
	psCmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(
			`[void][Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]; `+
				`$v = New-Object Windows.Security.Credentials.PasswordVault; `+
				`try { $c = $v.Retrieve('%s','%s'); $v.Remove($c) } catch { exit 1 }`,
			serviceName, key,
		),
	)
	if _, err := psCmd.CombinedOutput(); err != nil {
		return ErrNotFound
	}
	return nil
}
