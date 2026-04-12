package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"scuffinger/internal/logging"
)

// DebugHandler provides REST endpoints for browsing the PostgreSQL database
// and ValKey (Redis) cache — intended for development and debugging.
type DebugHandler struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	log  *logging.Logger
}

// NewDebugHandler creates a new DebugHandler.
func NewDebugHandler(pool *pgxpool.Pool, rdb *redis.Client, log *logging.Logger) *DebugHandler {
	return &DebugHandler{pool: pool, rdb: rdb, log: log}
}

// RegisterRoutes implements RouteRegistrar.
func (h *DebugHandler) RegisterRoutes(r *gin.Engine) {
	debug := r.Group("/api/debug")
	{
		// PostgreSQL
		pg := debug.Group("/pg")
		{
			pg.GET("/databases", h.ListDatabases)
			pg.GET("/tables", h.ListTables)
			pg.GET("/tables/:table/columns", h.DescribeTable)
			pg.GET("/tables/:table/rows", h.QueryRows)
		}

		// ValKey / Redis
		cache := debug.Group("/cache")
		{
			cache.GET("/keys", h.ScanKeys)
			cache.GET("/keys/*key", h.GetKey)
			cache.GET("/stats", h.CacheStats)
		}
	}
}

// ── PostgreSQL handlers ──────────────────────────────────────────────────────

// ListDatabases returns all non-template database names.
func (h *DebugHandler) ListDatabases(c *gin.Context) {
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		names = append(names, name)
	}
	c.JSON(http.StatusOK, gin.H{"databases": names})
}

// ListTables returns all tables in the given schema (default "public").
func (h *DebugHandler) ListTables(c *gin.Context) {
	schema := c.DefaultQuery("schema", "public")

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT table_name, table_type
		 FROM information_schema.tables
		 WHERE table_schema = $1
		 ORDER BY table_name`, schema)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type tableInfo struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	var tables []tableInfo
	for rows.Next() {
		var t tableInfo
		if err := rows.Scan(&t.Name, &t.Type); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tables = append(tables, t)
	}
	c.JSON(http.StatusOK, gin.H{"schema": schema, "tables": tables})
}

// DescribeTable returns column metadata for a table.
func (h *DebugHandler) DescribeTable(c *gin.Context) {
	table := c.Param("table")
	schema := c.DefaultQuery("schema", "public")

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT column_name, data_type, is_nullable, column_default
		 FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position`, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type colInfo struct {
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Nullable string  `json:"nullable"`
		Default  *string `json:"default"`
	}
	var cols []colInfo
	for rows.Next() {
		var col colInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Default); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("table %q not found in schema %q", table, schema)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"schema": schema, "table": table, "columns": cols})
}

// QueryRows returns rows from a table with optional fuzzy search, date filtering,
// and pagination.
//
// Query parameters:
//   - q         — fuzzy search (ILIKE) across all text/varchar/jsonb columns
//   - from, to  — ISO 8601 date bounds applied to the first detected timestamp column
//   - column    — restrict date filter to a specific column name
//   - sort      — column to ORDER BY (default: first column)
//   - order     — "asc" or "desc" (default: "asc")
//   - limit     — max rows to return (default 50, max 500)
//   - offset    — pagination offset (default 0)
//   - schema    — schema name (default "public")
func (h *DebugHandler) QueryRows(c *gin.Context) {
	table := c.Param("table")
	schema := c.DefaultQuery("schema", "public")
	q := c.Query("q")
	fromStr := c.Query("from")
	toStr := c.Query("to")
	dateCol := c.Query("column")
	sortCol := c.Query("sort")
	order := strings.ToUpper(c.DefaultQuery("order", "asc"))
	if order != "ASC" && order != "DESC" {
		order = "ASC"
	}

	limit := clampInt(c.DefaultQuery("limit", "50"), 1, 500)
	offset := clampInt(c.DefaultQuery("offset", "0"), 0, 1_000_000)

	ctx := c.Request.Context()

	// ── Introspect columns ───────────────────────────────────────────
	colRows, err := h.pool.Query(ctx,
		`SELECT column_name, data_type
		 FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position`, schema, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer colRows.Close()

	type colMeta struct {
		Name string
		Type string
	}
	var columns []colMeta
	for colRows.Next() {
		var cm colMeta
		if err := colRows.Scan(&cm.Name, &cm.Type); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		columns = append(columns, cm)
	}
	if len(columns) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("table %q not found in schema %q", table, schema)})
		return
	}

	// Identify text-like columns for fuzzy search and timestamp columns for date filter.
	var textCols []string
	var tsCols []string
	for _, col := range columns {
		dt := strings.ToLower(col.Type)
		switch {
		case strings.Contains(dt, "char"), strings.Contains(dt, "text"), strings.Contains(dt, "jsonb"), strings.Contains(dt, "json"):
			textCols = append(textCols, col.Name)
		case strings.Contains(dt, "timestamp"), strings.Contains(dt, "date"):
			tsCols = append(tsCols, col.Name)
		}
	}

	// ── Build query ──────────────────────────────────────────────────
	// Use the quoted identifier form to prevent SQL injection via table/column names.
	quotedTable := pgQuoteIdent(schema) + "." + pgQuoteIdent(table)

	var conditions []string
	var args []any
	argN := 1

	// Fuzzy search across text columns
	if q != "" && len(textCols) > 0 {
		var ors []string
		for _, col := range textCols {
			ors = append(ors, fmt.Sprintf("%s::text ILIKE $%d", pgQuoteIdent(col), argN))
		}
		conditions = append(conditions, "("+strings.Join(ors, " OR ")+")")
		args = append(args, "%"+q+"%")
		argN++
	}

	// Date range filter
	tsTarget := ""
	if dateCol != "" {
		// Validate the requested column exists and is a timestamp
		for _, col := range tsCols {
			if strings.EqualFold(col, dateCol) {
				tsTarget = col
				break
			}
		}
	}
	if tsTarget == "" && len(tsCols) > 0 {
		tsTarget = tsCols[0]
	}

	if tsTarget != "" {
		if fromStr != "" {
			if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
				conditions = append(conditions, fmt.Sprintf("%s >= $%d", pgQuoteIdent(tsTarget), argN))
				args = append(args, t)
				argN++
			} else if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				conditions = append(conditions, fmt.Sprintf("%s >= $%d", pgQuoteIdent(tsTarget), argN))
				args = append(args, t)
				argN++
			}
		}
		if toStr != "" {
			if t, err := time.Parse(time.RFC3339, toStr); err == nil {
				conditions = append(conditions, fmt.Sprintf("%s <= $%d", pgQuoteIdent(tsTarget), argN))
				args = append(args, t)
				argN++
			} else if t, err := time.Parse("2006-01-02", toStr); err == nil {
				// End-of-day inclusive
				conditions = append(conditions, fmt.Sprintf("%s <= $%d", pgQuoteIdent(tsTarget), argN))
				args = append(args, t.Add(24*time.Hour-time.Nanosecond))
				argN++
			}
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Sort column validation
	sortExpr := "1"
	if sortCol != "" {
		for _, col := range columns {
			if strings.EqualFold(col.Name, sortCol) {
				sortExpr = pgQuoteIdent(col.Name)
				break
			}
		}
	}

	// ── Count total matching rows ────────────────────────────────────
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", quotedTable, where)
	var total int64
	if err := h.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ── Fetch rows ───────────────────────────────────────────────────
	dataQuery := fmt.Sprintf("SELECT * FROM %s%s ORDER BY %s %s LIMIT %d OFFSET %d",
		quotedTable, where, sortExpr, order, limit, offset)

	dataRows, err := h.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer dataRows.Close()

	fieldDescs := dataRows.FieldDescriptions()
	var result []map[string]any
	for dataRows.Next() {
		values, err := dataRows.Values()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		row := make(map[string]any, len(fieldDescs))
		for i, fd := range fieldDescs {
			row[string(fd.Name)] = values[i]
		}
		result = append(result, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"schema": schema,
		"table":  table,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"rows":   result,
	})
}

// ── ValKey / Redis handlers ──────────────────────────────────────────────────

// ScanKeys scans cache keys with an optional glob pattern and fuzzy filter.
//
// Query parameters:
//   - pattern — Redis SCAN glob (default "*")
//   - q       — substring filter applied to key names (case-insensitive)
//   - limit   — max keys to return (default 50, max 500)
//   - cursor  — SCAN cursor for pagination (default "0")
//   - type    — filter by Redis key type (string, list, set, hash, zset)
func (h *DebugHandler) ScanKeys(c *gin.Context) {
	pattern := c.DefaultQuery("pattern", "*")
	q := strings.ToLower(c.Query("q"))
	keyType := strings.ToLower(c.Query("type"))
	limit := clampInt(c.DefaultQuery("limit", "50"), 1, 500)
	cursorStr := c.DefaultQuery("cursor", "0")
	cursor, _ := strconv.ParseUint(cursorStr, 10, 64)

	ctx := c.Request.Context()

	type keyEntry struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		TTL  int64  `json:"ttl_seconds"` // -1 = no expiry, -2 = key gone
	}

	var keys []keyEntry
	nextCursor := cursor

	// Keep scanning until we have enough matches or the scan is exhausted.
	for len(keys) < limit {
		var scannedKeys []string
		var err error
		scannedKeys, nextCursor, err = h.rdb.Scan(ctx, nextCursor, pattern, int64(limit*2)).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, k := range scannedKeys {
			if len(keys) >= limit {
				break
			}

			// Fuzzy filter on key name
			if q != "" && !strings.Contains(strings.ToLower(k), q) {
				continue
			}

			// Get type
			kt, err := h.rdb.Type(ctx, k).Result()
			if err != nil {
				continue
			}

			// Filter by type
			if keyType != "" && kt != keyType {
				continue
			}

			// Get TTL
			ttl, err := h.rdb.TTL(ctx, k).Result()
			if err != nil {
				continue
			}
			ttlSec := int64(ttl.Seconds())
			if ttl < 0 {
				ttlSec = int64(ttl / time.Second) // preserves -1 and -2
			}

			keys = append(keys, keyEntry{
				Key:  k,
				Type: kt,
				TTL:  ttlSec,
			})
		}

		// If cursor returned to 0, the full scan is complete.
		if nextCursor == 0 {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pattern": pattern,
		"cursor":  strconv.FormatUint(nextCursor, 10),
		"count":   len(keys),
		"keys":    keys,
	})
}

// GetKey returns the value, type, and TTL for a single cache key.
// For complex types (list, set, hash, zset), it returns the full contents
// (capped at 1000 elements).
func (h *DebugHandler) GetKey(c *gin.Context) {
	// The wildcard param includes the leading /, so trim it.
	key := strings.TrimPrefix(c.Param("key"), "/")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	ctx := c.Request.Context()
	const maxElements = 1000

	// Check existence + type
	kt, err := h.rdb.Type(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if kt == "none" {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("key %q does not exist", key)})
		return
	}

	// TTL
	ttl, _ := h.rdb.TTL(ctx, key).Result()
	ttlSec := int64(ttl.Seconds())
	if ttl < 0 {
		ttlSec = int64(ttl / time.Second)
	}

	// Read value based on type
	var value any
	switch kt {
	case "string":
		value, err = h.rdb.Get(ctx, key).Result()
	case "list":
		value, err = h.rdb.LRange(ctx, key, 0, maxElements-1).Result()
	case "set":
		value, err = h.rdb.SMembers(ctx, key).Result()
	case "zset":
		value, err = h.rdb.ZRangeWithScores(ctx, key, 0, maxElements-1).Result()
	case "hash":
		value, err = h.rdb.HGetAll(ctx, key).Result()
	case "stream":
		msgs, e := h.rdb.XRange(ctx, key, "-", "+").Result()
		if e == nil && len(msgs) > maxElements {
			msgs = msgs[:maxElements]
		}
		value, err = msgs, e
	default:
		value = fmt.Sprintf("(unsupported type: %s)", kt)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Memory usage (approximate)
	mem, _ := h.rdb.MemoryUsage(ctx, key).Result()

	c.JSON(http.StatusOK, gin.H{
		"key":          key,
		"type":         kt,
		"ttl_seconds":  ttlSec,
		"memory_bytes": mem,
		"value":        value,
	})
}

// CacheStats returns ValKey/Redis server statistics.
func (h *DebugHandler) CacheStats(c *gin.Context) {
	ctx := c.Request.Context()

	info, err := h.rdb.Info(ctx, "server", "memory", "clients", "keyspace", "stats").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse INFO into structured sections.
	sections := parseRedisInfo(info)

	// Also return the DB size for convenience.
	dbSize, _ := h.rdb.DBSize(ctx).Result()

	c.JSON(http.StatusOK, gin.H{
		"db_size":  dbSize,
		"sections": sections,
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

// pgQuoteIdent quotes a PostgreSQL identifier to prevent SQL injection.
// It doubles any internal double-quotes per the SQL standard.
func pgQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// clampInt parses s as an integer and clamps it between lo and hi.
func clampInt(s string, lo, hi int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

// parseRedisInfo parses a Redis INFO response into a map of section → key/value pairs.
func parseRedisInfo(info string) map[string]map[string]string {
	sections := make(map[string]map[string]string)
	currentSection := "default"

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			currentSection = strings.ToLower(strings.TrimPrefix(line, "# "))
			if _, ok := sections[currentSection]; !ok {
				sections[currentSection] = make(map[string]string)
			}
			continue
		}
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			if _, ok := sections[currentSection]; !ok {
				sections[currentSection] = make(map[string]string)
			}
			sections[currentSection][parts[0]] = parts[1]
		}
	}
	return sections
}
