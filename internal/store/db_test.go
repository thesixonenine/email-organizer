package store

import (
	"os"
	"testing"
)

func TestInitDB(t *testing.T) {
	f, _ := os.CreateTemp("", "test-*.db")
	defer os.Remove(f.Name())
	f.Close()

	db, err := InitDB(f.Name())
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	rows, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	expected := []string{"mailboxes", "messages", "rule_logs", "rules"}
	for _, e := range expected {
		found := false
		for _, t := range tables {
			if t == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("table %s missing", e)
		}
	}
}