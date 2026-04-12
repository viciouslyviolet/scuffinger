// Package selftest embeds all SQL queries used by the startup self-test.
// Each query set lives in its own sub-directory under database/.
package selftest

import _ "embed"

// ── DDL / admin queries (use fmt.Sprintf to inject identifiers) ──────────────

//go:embed terminate_connections.sql
var TerminateConnections string

//go:embed drop_database.sql
var DropDatabase string

//go:embed create_database.sql
var CreateDatabase string

// ── Self-test CRUD queries ───────────────────────────────────────────────────

//go:embed create_self_test_table.sql
var CreateSelfTestTable string

//go:embed insert_self_test.sql
var InsertSelfTest string

//go:embed select_self_test.sql
var SelectSelfTest string

//go:embed update_self_test.sql
var UpdateSelfTest string
