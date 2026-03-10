package audit

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"
)

// Logger writes action audit entries to SQLite.
type Logger struct {
	db *sql.DB
}

// Entry represents a single audit log record.
type Entry struct {
	GitHubUser string
	Action     string
	Parameters map[string]string
	Result     string
	IPAddress  string
}

// New opens (or creates) the SQLite database and ensures the schema exists.
func New(dbPath string) (*Logger, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		github_user TEXT,
		action TEXT,
		parameters TEXT,
		result TEXT,
		ip_address TEXT
	)`)
	if err != nil {
		return nil, err
	}

	return &Logger{db: db}, nil
}

// Log inserts an audit entry.
func (l *Logger) Log(entry Entry) error {
	params, err := json.Marshal(entry.Parameters)
	if err != nil {
		return err
	}

	_, err = l.db.Exec(
		`INSERT INTO audit_logs (timestamp, github_user, action, parameters, result, ip_address)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now().UTC(),
		entry.GitHubUser,
		entry.Action,
		string(params),
		entry.Result,
		entry.IPAddress,
	)
	return err
}

// Close closes the database connection.
func (l *Logger) Close() error {
	return l.db.Close()
}
