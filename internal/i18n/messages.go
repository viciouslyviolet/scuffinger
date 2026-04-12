// Package i18n provides a string dictionary for all user-facing messages.
// To add a new language, create a new Messages map and call Set().
package i18n

import "fmt"

// Key identifies a translatable message.
type Key string

// ── Message keys ─────────────────────────────────────────────────────────────

const (
	// Config
	MsgConfigLoaded Key = "config.loaded"
	ErrConfigLoad   Key = "err.config.load"

	// Bootstrap
	MsgBootstrapConnecting  Key = "bootstrap.connecting"
	MsgBootstrapSelfTests   Key = "bootstrap.self_tests"
	MsgBootstrapTestsPassed Key = "bootstrap.tests_passed"
	MsgBootstrapHealthStart Key = "bootstrap.health_start"
	ErrBootstrapConnect     Key = "err.bootstrap.connect"
	ErrBootstrapSelfTests   Key = "err.bootstrap.self_tests"

	// Manager
	MsgManagerConnecting     Key = "manager.connecting"
	MsgManagerConnected      Key = "manager.connected"
	MsgManagerSelfTest       Key = "manager.self_test"
	MsgManagerSelfTestPassed Key = "manager.self_test_passed"
	MsgManagerClosing        Key = "manager.closing"
	WarnManagerHealthFailed  Key = "manager.health_failed"
	ErrManagerConnect        Key = "err.manager.connect"
	ErrManagerSelfTest       Key = "err.manager.self_test"
	ErrManagerClose          Key = "err.manager.close"
	ErrManagerShutdown       Key = "err.manager.shutdown"

	// Cache service
	MsgCacheSelfTestInit   Key = "cache.self_test_init"
	MsgCacheSelfTestPassed Key = "cache.self_test_passed"
	ErrCachePing           Key = "err.cache.ping"
	ErrCacheSetInit        Key = "err.cache.set_init"
	ErrCacheGetInit        Key = "err.cache.get_init"
	ErrCacheInitMismatch   Key = "err.cache.init_mismatch"

	// Database service
	MsgDbCreatingTestDb  Key = "db.creating_testdb"
	MsgDbRunningCrud     Key = "db.running_crud"
	MsgDbSelfTestPassed  Key = "db.self_test_passed"
	WarnDbDropTestFailed Key = "db.drop_test_failed"
	ErrDbConnect         Key = "err.db.connect"
	ErrDbPing            Key = "err.db.ping"
	ErrDbCreateTestDb    Key = "err.db.create_testdb"
	ErrDbConnectTestDb   Key = "err.db.connect_testdb"
	ErrDbCrud            Key = "err.db.crud"
	ErrDbCreateTable     Key = "err.db.create_table"
	ErrDbInsert          Key = "err.db.insert"
	ErrDbRead            Key = "err.db.read"
	ErrDbReadMismatch    Key = "err.db.read_mismatch"
	ErrDbUpdate          Key = "err.db.update"
	ErrDbReadAfterUpdate Key = "err.db.read_after_update"
	ErrDbUpdateMismatch  Key = "err.db.update_mismatch"

	// Server
	MsgServerStarting Key = "server.starting"
	MsgServerShutdown Key = "server.shutdown"
	MsgServerStopped  Key = "server.stopped"
	MsgServerRoutes   Key = "server.routes_registered"
	ErrServerListen   Key = "err.server.listen"
	ErrServerShutdown Key = "err.server.shutdown"

	// Health
	MsgHealthReady    Key = "health.ready"
	MsgHealthNotReady Key = "health.not_ready"

	// GitHub service
	MsgGhConnecting      Key = "gh.connecting"
	MsgGhConnected       Key = "gh.connected"
	MsgGhAuthToken       Key = "gh.auth_token"
	MsgGhAuthApp         Key = "gh.auth_app"
	MsgGhSelfTestPassed  Key = "gh.self_test_passed"
	MsgGhRateRemaining   Key = "gh.rate_remaining"
	WarnGhRateLow        Key = "gh.rate_low"
	ErrGhNotConfigured   Key = "err.gh.not_configured"
	ErrGhAuth            Key = "err.gh.auth"
	ErrGhReadKey         Key = "err.gh.read_key"
	ErrGhParseKey        Key = "err.gh.parse_key"
	ErrGhInstallToken    Key = "err.gh.install_token"
	ErrGhSelfTest        Key = "err.gh.self_test"
	ErrGhRateLimit       Key = "err.gh.rate_limit"
	ErrGhFetchUser       Key = "err.gh.fetch_user"
	ErrGhFetchOrg        Key = "err.gh.fetch_org"
	ErrGhFetchRepo       Key = "err.gh.fetch_repo"
	ErrGhFetchBranches   Key = "err.gh.fetch_branches"
	ErrGhFetchWorkflows  Key = "err.gh.fetch_workflows"
	ErrGhFetchRuns       Key = "err.gh.fetch_runs"
	ErrGhFetchRateLimit  Key = "err.gh.fetch_rate_limit"
	ErrGhInvalidWorkflow Key = "err.gh.invalid_workflow"

	// GitHub collector
	MsgGhCollectorStarting         Key = "gh_collector.starting"
	MsgGhCollectorTick             Key = "gh_collector.tick"
	MsgGhCollectorRepo             Key = "gh_collector.repo"
	MsgGhCollectorStopped          Key = "gh_collector.stopped"
	MsgGhCollectorPassed           Key = "gh_collector.self_test_passed"
	MsgGhCollectorAnnotations      Key = "gh_collector.annotations"
	ErrGhCollectorFetchRepo        Key = "err.gh_collector.fetch_repo"
	ErrGhCollectorFetchRuns        Key = "err.gh_collector.fetch_runs"
	ErrGhCollectorFetchJobs        Key = "err.gh_collector.fetch_jobs"
	ErrGhCollectorFetchAnnotations Key = "err.gh_collector.fetch_annotations"
	MsgGhCollectorLockAcquired     Key = "gh_collector.lock_acquired"
	MsgGhCollectorLockSkipped      Key = "gh_collector.lock_skipped"
	WarnGhCollectorLockError       Key = "gh_collector.lock_error"
	ErrGhCollectorNoRepos          Key = "err.gh_collector.no_repos"
	ErrGhCollectorParseRepo        Key = "err.gh_collector.parse_repo"

	// GitHub OAuth auth
	MsgAuthDevicePrompt   Key = "auth.device_prompt"
	MsgAuthPolling        Key = "auth.polling"
	MsgAuthSuccess        Key = "auth.success"
	MsgAuthLoggedOut      Key = "auth.logged_out"
	MsgAuthStatusLoggedIn Key = "auth.status_logged_in"
	MsgAuthStatusNoToken  Key = "auth.status_no_token"
	MsgAuthTokenFromVault Key = "auth.token_from_vault"
	ErrAuthNoClientID     Key = "err.auth.no_client_id"
	ErrAuthDeviceCode     Key = "err.auth.device_code"
	ErrAuthPoll           Key = "err.auth.poll"
	ErrAuthSaveToken      Key = "err.auth.save_token"
	ErrAuthVerifyToken    Key = "err.auth.verify_token"

	// CLI commands
	CmdRootShort          Key = "cmd.root.short"
	CmdRootLong           Key = "cmd.root.long"
	CmdVersionShort       Key = "cmd.version.short"
	CmdVersionLong        Key = "cmd.version.long"
	CmdServeShort         Key = "cmd.serve.short"
	CmdServeLong          Key = "cmd.serve.long"
	CmdGitHubShort        Key = "cmd.github.short"
	CmdGitHubLong         Key = "cmd.github.long"
	CmdGitHubAuthShort    Key = "cmd.github_auth.short"
	CmdGitHubAuthLong     Key = "cmd.github_auth.long"
	CmdGitHubStatusShort  Key = "cmd.github_status.short"
	CmdGitHubStatusLong   Key = "cmd.github_status.long"
	CmdGitHubLogoutShort  Key = "cmd.github_logout.short"
	CmdGitHubLogoutLong   Key = "cmd.github_logout.long"
	CmdGitHubMonitorShort Key = "cmd.github_monitor.short"
	CmdGitHubMonitorLong  Key = "cmd.github_monitor.long"
	CmdFlagConfig         Key = "cmd.flag.config"
)

// Messages maps keys to translated strings.
type Messages map[Key]string

// ── Active language ──────────────────────────────────────────────────────────

var active = En

// Set switches the active message set to the given language.
func Set(m Messages) {
	active = m
}

// Get returns the translated string for the given key.
// Falls back to the key name if no translation is found.
func Get(key Key) string {
	if msg, ok := active[key]; ok {
		return msg
	}
	return string(key)
}

// Err creates a formatted error using a translated message prefix and wrapping cause.
//
//	return i18n.Err(i18n.ErrCachePing, err)
//	// → "Cache is not reachable: dial tcp: connection refused"
func Err(key Key, cause error) error {
	return fmt.Errorf("%s: %w", Get(key), cause)
}
