package bolt

import (
	"github.com/boltdb/bolt"
	"net/http"
	"strconv"
)

type DB struct {
	*bolt.DB
}

func (db *DB) ReturnBackup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.View(func(tx *bolt.Tx) error {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="micromdm.db"`)
			w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
			_, err := tx.WriteTo(w)
			return err
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
