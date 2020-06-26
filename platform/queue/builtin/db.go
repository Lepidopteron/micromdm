package builtin

import (
	"context"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/micromdm/micromdm/platform/queue"
	"github.com/pkg/errors"

	"github.com/micromdm/micromdm/mdm"
)

const (
	DeviceCommandBucket = "mdm.DeviceCommands"
)

type DB struct {
	*bolt.DB
}

func NewDB(db *bolt.DB) (*DB, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(DeviceCommandBucket))
		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "creating %s bucket", DeviceCommandBucket)
	}
	return &DB{DB: db}, nil
}

func (db *DB) Save(ctx context.Context, cmd *queue.DeviceCommand) error {
	tx, err := db.DB.Begin(true)
	if err != nil {
		return errors.Wrap(err, "begin transaction")
	}
	bkt := tx.Bucket([]byte(DeviceCommandBucket))
	if bkt == nil {
		return fmt.Errorf("bucket %q not found!", DeviceCommandBucket)
	}
	devproto, err := queue.MarshalDeviceCommand(cmd)
	if err != nil {
		return errors.Wrap(err, "marshalling DeviceCommand")
	}
	key := []byte(cmd.DeviceUDID)
	if err := bkt.Put(key, devproto); err != nil {
		return errors.Wrap(err, "put DeviceCommand to boltdb")
	}
	return tx.Commit()
}

func (db *DB) DeviceCommand(ctx context.Context, udid string) (*queue.DeviceCommand, error) {
	var dev queue.DeviceCommand
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(DeviceCommandBucket))
		v := b.Get([]byte(udid))
		if v == nil {
			return &queue.NotFound{ResourceType: "DeviceCommand", Message: fmt.Sprintf("udid %s", udid)}
		}
		return queue.UnmarshalDeviceCommand(v, &dev)
	})
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (db *DB) UpdateCommandStatus(ctx context.Context, resp mdm.Response) error {
	// This method is implemented to fulfill the Store interface. It is not used by this datastore.
	return nil
}