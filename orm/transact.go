// Package dao is the data access object swiss army knife / "black box".
package orm

import (
	"context"
	"database/sql"
	//"github.com/rbastic/dyndao/
	"fmt"
	log15 "gopkg.in/inconshreveable/log15.v2"
	"runtime/debug"
)

type TxFuncType func(*sql.Tx) error

// Transact is meant to group operations into transactions, simplify error
// handling, and recover from any panics.  See:
// http://stackoverflow.com/questions/16184238/database-sql-tx-detecting-commit-or-rollback
// Please note this function has been changed from the above post to use
// contexts
func (o *ORM) Transact(ctx context.Context, txFunc TxFuncType) error {
	tx, err := o.RawConn.BeginTx(ctx, nil)
	if err != nil {
		log15.Error("[Transact]", "BeginTx", err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			default:
				err = fmt.Errorf("%s", p)
			}

			err = fmt.Errorf("%s [%s]", err, debug.Stack())
			log15.Error("Transact", "defer_panic_error", err.Error())
		}
		if err != nil {
			rollbackErr := tx.Rollback()

			// TODO: If rollback has an error, does
			// this mean the code below executes
			// and we end up with an additional
			// error?
			if rollbackErr != nil {
				log15.Error("Transact", "defer_Rollback_error", rollbackErr)
			}
			return
		}
		err = tx.Commit()
		if err != nil {
			log15.Error("Transact", "defer_Commit_error", err)
		}
	}()

	err = txFunc(tx)

	if err != nil {
		log15.Error("Transact", "error_in_txFunc", err)
		return err
	}

	return nil
}
