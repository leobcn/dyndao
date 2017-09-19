package orm

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
)

func prepareAndExecSQL(db *sql.DB, sqlStr string) (sql.Result, error) {
	stmt, err := db.PrepareContext(context.TODO(), sqlStr)
	if err != nil {
		return nil, errors.Wrap(err, "prepareAndExecSQL/PrepareContext ("+sqlStr+")")
	}
	defer func() {
		stmtErr := stmt.Close()
		if stmtErr != nil {
			fmt.Println(stmtErr) // TODO: logging implementation
		}
	}()
	r, err := stmt.ExecContext(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "prepareAndExecSQL/ExecContext ("+sqlStr+")")
	}
	return r, nil
}