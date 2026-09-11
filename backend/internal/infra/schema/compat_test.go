package schema

import (
	"os"
	"strings"
	"testing"
)

// Every statement must be re-runnable: the reconciliation is executed on every deploy, including the
// first one after a database has already been fixed.
func TestLegacyConstraintStatementsAreIdempotent(t *testing.T) {
	for _, stmt := range LegacyConstraintStatements() {
		if !strings.Contains(stmt, "IF EXISTS") {
			t.Errorf("statement is not idempotent, missing IF EXISTS: %s", stmt)
		}
		if !strings.HasPrefix(stmt, "ALTER TABLE ") || !strings.Contains(stmt, "DROP CONSTRAINT") {
			t.Errorf("unexpected statement shape: %s", stmt)
		}
	}
}

// The list drives a DROP, so a typo in a table or column name would silently do nothing (the guard is
// IF EXISTS) and leave the migration broken on the next run. Pin the full set.
func TestLegacyConstraintStatementsCoverEveryAffectedColumn(t *testing.T) {
	// Derived from the original raw-SQL migrations: they declared these UNIQUE, so PostgreSQL named the
	// constraints <table>_<column>_key, while the models declare them with `uniqueIndex`.
	want := []string{
		"users_phone_key",
		"users_email_key",
		"users_wechat_openid_key",
		"links_short_code_key",
		"workspaces_slug_key",
		"api_tokens_token_hash_key",
		"click_logs_event_id_key",
		"domains_user_id_name_key",
	}

	got := make(map[string]bool, len(want))
	for _, stmt := range LegacyConstraintStatements() {
		for _, name := range want {
			if strings.Contains(stmt, name) {
				got[name] = true
			}
		}
	}

	for _, name := range want {
		if !got[name] {
			t.Errorf("no statement drops the legacy constraint %s", name)
		}
	}
}

// The reconciliation targets PostgreSQL's generated constraint names, so it must not run any DDL on a
// different engine. Pinning the guard keeps a non-Postgres deployment from failing every migration on
// syntax it does not understand.
func TestReconcileIsPostgresOnly(t *testing.T) {
	src, err := os.ReadFile("compat.go")
	if err != nil {
		t.Fatalf("cannot read compat.go: %v", err)
	}
	body := string(src)

	guard := `if db.Name() != "postgres"`
	if !strings.Contains(body, guard) {
		t.Errorf("the schema helpers must return early for non-Postgres engines (expected %s)", guard)
	}
	// The guard has to come before any DDL is executed, otherwise it is useless.
	if strings.Index(body, guard) > strings.Index(body, "db.Exec(") {
		t.Error("the engine guard appears after the first db.Exec call")
	}
	if n := strings.Count(body, guard); n < 2 {
		t.Errorf("expected the engine guard on every DDL helper, found %d", n)
	}
}

// AutoMigrate can create tables and columns but not types, so a model field tagged `type:<something>`
// only works if that type is created beforehand. Forgetting one fails the whole migration on a database
// that has never run the original SQL migrations, which is how the click_platform enum was missed.
func TestEveryEnumColumnHasACreatedType(t *testing.T) {
	// The enum types the entity package references through a `type:` tag.
	referenced := map[string]bool{
		"click_platform": true,
	}

	created := make(map[string]bool)
	for _, stmt := range EnumTypeStatements() {
		for name := range referenced {
			if strings.Contains(stmt, "CREATE TYPE "+name) {
				created[name] = true
			}
		}
	}

	for name := range referenced {
		if !created[name] {
			t.Errorf("the models use the enum type %q but no statement creates it; "+
				"AutoMigrate cannot create types, so a fresh database will fail", name)
		}
	}
}

// Enum statements run on every start, so a plain CREATE TYPE would fail the second time. The DO block
// swallows duplicate_object instead.
func TestEnumTypeStatementsAreIdempotent(t *testing.T) {
	statements := EnumTypeStatements()
	if len(statements) == 0 {
		t.Fatal("expected at least one enum type statement")
	}
	for _, stmt := range statements {
		if !strings.Contains(stmt, "duplicate_object") {
			t.Errorf("statement is not idempotent, it does not handle duplicate_object: %s", stmt)
		}
		if !strings.Contains(stmt, "CREATE TYPE ") {
			t.Errorf("unexpected statement shape: %s", stmt)
		}
	}
}
