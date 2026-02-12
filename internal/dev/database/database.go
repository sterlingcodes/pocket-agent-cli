package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3" // sqlite3 driver registration
	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// QueryResult holds the results of a SQL query
type QueryResult struct {
	Database string                   `json:"database"`
	Query    string                   `json:"query"`
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"row_count"`
}

// SchemaResult holds database schema information
type SchemaResult struct {
	Database string  `json:"database"`
	Tables   []Table `json:"tables"`
}

// Table holds table metadata
type Table struct {
	Name     string   `json:"name"`
	Columns  []Column `json:"columns"`
	RowCount int64    `json:"row_count"`
}

// Column holds column metadata
type Column struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	NotNull      bool        `json:"not_null"`
	PrimaryKey   bool        `json:"primary_key"`
	DefaultValue interface{} `json:"default_value"`
}

// TablesResult holds a quick list of tables
type TablesResult struct {
	Database string      `json:"database"`
	Tables   []TableInfo `json:"tables"`
	Count    int         `json:"count"`
}

// TableInfo holds basic table information
type TableInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// NewCmd returns the database command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "db",
		Aliases: []string{"database", "sql", "sqlite"},
		Short:   "SQLite database commands",
	}

	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newSchemaCmd())
	cmd.AddCommand(newTablesCmd())

	return cmd
}

func newQueryCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "query [db-path] [sql]",
		Short: "Execute a SELECT query and return results as JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]
			sqlQuery := args[1]

			if err := validateDBPath(dbPath); err != nil {
				return err
			}

			if err := validateReadOnly(sqlQuery); err != nil {
				return err
			}

			// Append LIMIT if not already present
			upperQuery := strings.ToUpper(sqlQuery)
			if !strings.Contains(upperQuery, "LIMIT") {
				sqlQuery = fmt.Sprintf("%s LIMIT %d", sqlQuery, limit)
			}

			db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
			if err != nil {
				return output.PrintError("db_open_failed", fmt.Sprintf("Failed to open database: %s", err.Error()), nil)
			}
			defer db.Close()

			rows, err := db.Query(sqlQuery)
			if err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Query failed: %s", err.Error()), nil)
			}
			defer rows.Close()

			columns, err := rows.Columns()
			if err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Failed to get columns: %s", err.Error()), nil)
			}

			var resultRows []map[string]interface{}

			for rows.Next() {
				values := make([]sql.RawBytes, len(columns))
				scanArgs := make([]interface{}, len(columns))
				for i := range values {
					scanArgs[i] = &values[i]
				}

				if err := rows.Scan(scanArgs...); err != nil {
					return output.PrintError("scan_failed", fmt.Sprintf("Failed to scan row: %s", err.Error()), nil)
				}

				row := make(map[string]interface{})
				for i, col := range columns {
					if values[i] == nil {
						row[col] = nil
					} else {
						row[col] = string(values[i])
					}
				}
				resultRows = append(resultRows, row)
			}

			if err := rows.Err(); err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Row iteration error: %s", err.Error()), nil)
			}

			if resultRows == nil {
				resultRows = []map[string]interface{}{}
			}

			result := QueryResult{
				Database: dbPath,
				Query:    sqlQuery,
				Columns:  columns,
				Rows:     resultRows,
				RowCount: len(resultRows),
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum number of rows to return")

	return cmd
}

func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema [db-path]",
		Short: "List all tables and their columns",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]

			if err := validateDBPath(dbPath); err != nil {
				return err
			}

			db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
			if err != nil {
				return output.PrintError("db_open_failed", fmt.Sprintf("Failed to open database: %s", err.Error()), nil)
			}
			defer db.Close()

			// Get all table names
			tableRows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
			if err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Failed to list tables: %s", err.Error()), nil)
			}
			defer tableRows.Close()

			var tableNames []string
			for tableRows.Next() {
				var name string
				if err := tableRows.Scan(&name); err != nil {
					return output.PrintError("scan_failed", fmt.Sprintf("Failed to scan table name: %s", err.Error()), nil)
				}
				tableNames = append(tableNames, name)
			}
			if err := tableRows.Err(); err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Table iteration error: %s", err.Error()), nil)
			}

			var tables []Table
			for _, tableName := range tableNames {
				table := Table{Name: tableName}

				// Get column info using PRAGMA
				pragmaQuery := fmt.Sprintf("PRAGMA table_info(\"%s\")", strings.ReplaceAll(tableName, "\"", "\"\"")) //nolint:gocritic // SQL syntax requires this format
				colRows, err := db.Query(pragmaQuery)
				if err != nil {
					return output.PrintError("query_failed", fmt.Sprintf("Failed to get columns for %s: %s", tableName, err.Error()), nil)
				}
				defer colRows.Close() //nolint:gocritic // rows must be closed after each query iteration

				for colRows.Next() {
					var cid int
					var name, colType string
					var notNull, pk int
					var dfltValue sql.NullString

					if err := colRows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
						return output.PrintError("scan_failed", fmt.Sprintf("Failed to scan column info: %s", err.Error()), nil)
					}

					col := Column{
						Name:       name,
						Type:       colType,
						NotNull:    notNull == 1,
						PrimaryKey: pk > 0,
					}

					if dfltValue.Valid {
						col.DefaultValue = dfltValue.String
					}

					table.Columns = append(table.Columns, col)
				}
				if err := colRows.Err(); err != nil {
					return output.PrintError("query_failed", fmt.Sprintf("Column iteration error: %s", err.Error()), nil)
				}

				// Get row count
				countQuery := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", strings.ReplaceAll(tableName, "\"", "\"\"")) //nolint:gosec,gocritic // tableName comes from sqlite_master, not user input; SQL syntax requires this format
				var count int64
				if err := db.QueryRow(countQuery).Scan(&count); err == nil {
					table.RowCount = count
				}

				tables = append(tables, table)
			}

			result := SchemaResult{
				Database: dbPath,
				Tables:   tables,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newTablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tables [db-path]",
		Short: "Quick list of tables and views",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]

			if err := validateDBPath(dbPath); err != nil {
				return err
			}

			db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
			if err != nil {
				return output.PrintError("db_open_failed", fmt.Sprintf("Failed to open database: %s", err.Error()), nil)
			}
			defer db.Close()

			rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type IN ('table','view') ORDER BY name")
			if err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Failed to list tables: %s", err.Error()), nil)
			}
			defer rows.Close()

			var tables []TableInfo
			for rows.Next() {
				var name, tableType string
				if err := rows.Scan(&name, &tableType); err != nil {
					return output.PrintError("scan_failed", fmt.Sprintf("Failed to scan table info: %s", err.Error()), nil)
				}
				tables = append(tables, TableInfo{
					Name: name,
					Type: tableType,
				})
			}
			if err := rows.Err(); err != nil {
				return output.PrintError("query_failed", fmt.Sprintf("Table iteration error: %s", err.Error()), nil)
			}

			if tables == nil {
				tables = []TableInfo{}
			}

			result := TablesResult{
				Database: dbPath,
				Tables:   tables,
				Count:    len(tables),
			}

			return output.Print(result)
		},
	}

	return cmd
}

func validateDBPath(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return output.PrintError("file_not_found", fmt.Sprintf("Database file not found: %s", dbPath), nil)
	}
	return nil
}

func validateReadOnly(query string) error {
	// Normalize whitespace and convert to uppercase for checking
	normalized := strings.TrimSpace(strings.ToUpper(query))

	// Remove leading comments
	for strings.HasPrefix(normalized, "--") {
		idx := strings.Index(normalized, "\n")
		if idx == -1 {
			break
		}
		normalized = strings.TrimSpace(normalized[idx+1:])
	}

	// Allow SELECT, PRAGMA, and EXPLAIN
	allowed := false
	for _, prefix := range []string{"SELECT", "PRAGMA", "EXPLAIN"} {
		if strings.HasPrefix(normalized, prefix) {
			allowed = true
			break
		}
	}

	if !allowed {
		return output.PrintError("read_only", "Only SELECT, PRAGMA, and EXPLAIN statements are allowed", map[string]interface{}{
			"hint": "This tool is read-only. INSERT, UPDATE, DELETE, DROP, CREATE, and ALTER are not permitted.",
		})
	}

	// Additional check: reject dangerous statements even within subqueries
	dangerous := []string{
		"INSERT INTO", "UPDATE ", "DELETE FROM", "DROP TABLE", "DROP VIEW",
		"DROP INDEX", "DROP TRIGGER", "CREATE TABLE", "CREATE VIEW",
		"CREATE INDEX", "CREATE TRIGGER", "ALTER TABLE", "ATTACH ",
		"DETACH ",
	}
	for _, d := range dangerous {
		if strings.Contains(normalized, d) {
			return output.PrintError("read_only", fmt.Sprintf("Statement contains disallowed operation: %s", d), map[string]interface{}{
				"hint": "This tool is read-only. Modification statements are not permitted.",
			})
		}
	}

	return nil
}
