package datastore

import (
	"errors"
	"github.com/micromdm/micromdm/datastore/bolt"
	"github.com/micromdm/micromdm/datastore/mysql"
	"github.com/micromdm/micromdm/datastore/schema"
	"github.com/micromdm/micromdm/platform/pubsub"
)

func Create(id schema.ImplementationID, configMap map[string]string, pubClient pubsub.PublishSubscriber) (*schema.Schema, error) {
	if id == schema.Mysql {
		return mysql.Create(configMap, pubClient)
	} else if id == schema.Bolt {
		return bolt.Create(configMap, pubClient)
	}

	return nil, errors.New("unrecognized datastore type: " + string(id))
}
