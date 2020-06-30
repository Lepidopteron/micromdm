package datastore

import (
	"errors"
	"github.com/micromdm/micromdm/datastore/bolt"
	"github.com/micromdm/micromdm/datastore/mysql"
	"github.com/micromdm/micromdm/datastore/schema"
	"github.com/micromdm/micromdm/platform/pubsub"
)

func Create(id schema.ImplementationID, settingsMap map[string]string, pubClient pubsub.PublishSubscriber) (*schema.Schema, error) {
	if id == schema.Mysql {
		return mysql.Create(settingsMap, pubClient)
	} else if id == schema.Bolt {
		return bolt.Create(settingsMap, pubClient)
	}

	return nil, errors.New("unrecognized datastore type: " + string(id))
}
