package server

import (
	"testing"
)

func TestPgQuoteIdent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"public", `"public"`},
		{"my_table", `"my_table"`},
		{`evil"name`, `"evil""name"`},
		{"", `""`},
		{"CamelCase", `"CamelCase"`},
	}
	for _, tt := range tests {
		got := pgQuoteIdent(tt.input)
		if got != tt.want {
			t.Errorf("pgQuoteIdent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		s    string
		lo   int
		hi   int
		want int
	}{
		{"50", 1, 500, 50},
		{"0", 1, 500, 1},
		{"600", 1, 500, 500},
		{"abc", 1, 500, 1},
		{"-5", 0, 100, 0},
		{"100", 0, 100, 100},
	}
	for _, tt := range tests {
		got := clampInt(tt.s, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clampInt(%q, %d, %d) = %d, want %d", tt.s, tt.lo, tt.hi, got, tt.want)
		}
	}
}

func TestParseRedisInfo(t *testing.T) {
	info := `# Server
redis_version:7.2.4
redis_mode:standalone

# Memory
used_memory:1024
used_memory_human:1K

# Keyspace
db0:keys=5,expires=2,avg_ttl=300000
`

	sections := parseRedisInfo(info)

	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}

	if v, ok := sections["server"]["redis_version"]; !ok || v != "7.2.4" {
		t.Errorf("server.redis_version = %q, want %q", v, "7.2.4")
	}
	if v, ok := sections["memory"]["used_memory"]; !ok || v != "1024" {
		t.Errorf("memory.used_memory = %q, want %q", v, "1024")
	}
	if v, ok := sections["keyspace"]["db0"]; !ok {
		t.Error("keyspace.db0 not found")
	} else if v != "keys=5,expires=2,avg_ttl=300000" {
		t.Errorf("keyspace.db0 = %q", v)
	}
}
