package types

import (
	"database/sql"
	"fmt"
)

var txIsolationLevels = map[string]sql.IsolationLevel{
	"Default":         sql.LevelDefault,
	"ReadUncommitted": sql.LevelReadUncommitted,
	"ReadCommitted":   sql.LevelReadCommitted,
	"WriteCommitted":  sql.LevelWriteCommitted,
	"RepeatableRead":  sql.LevelRepeatableRead,
	"Snapshot":        sql.LevelSnapshot,
	"Serializable":    sql.LevelSerializable,
	"Linearizable":    sql.LevelLinearizable,
}

type IsolationLevel sql.IsolationLevel

func (t *IsolationLevel) UnmarshalMapstructure(value any) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("isolation_level: invalid type: expected string, got %T", value)
	}

	if level, ok := txIsolationLevels[s]; ok {
		*t = IsolationLevel(level)
		return nil
	}
	return fmt.Errorf("isolation_level: invalid value: %s", s)
}

func (t IsolationLevel) Unwrap() sql.IsolationLevel {
	return sql.IsolationLevel(t)
}
