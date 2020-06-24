package bolt

import (
	"github.com/boltdb/bolt"
	"github.com/micromdm/micromdm/datastore/schema"
	apnsbuiltin "github.com/micromdm/micromdm/platform/apns/builtin"
	blueprintbuiltin "github.com/micromdm/micromdm/platform/blueprint/builtin"
	configbuiltin "github.com/micromdm/micromdm/platform/config/builtin"
	syncbuiltin "github.com/micromdm/micromdm/platform/dep/sync/builtin"
	devicebuiltin "github.com/micromdm/micromdm/platform/device/builtin"
	profilebuiltin "github.com/micromdm/micromdm/platform/profile/builtin"
	"github.com/micromdm/micromdm/platform/pubsub"
	blockbuiltin "github.com/micromdm/micromdm/platform/remove/builtin"
	scepbuiltin "github.com/micromdm/micromdm/platform/scep/builtin"
	userbuiltin "github.com/micromdm/micromdm/platform/user/builtin"
	challengestore "github.com/micromdm/scep/challenge/bolt"
	"github.com/pkg/errors"
	stdlog "log"
	"path/filepath"
	"time"
)

func Create(configMap map[string]string, pubClient pubsub.PublishSubscriber) (*schema.Schema, error) {
	datastore := schema.Schema{
		ID: schema.Bolt,
	}

	// Database
	db, err := createDB(configMap)
	if err != nil {
		return nil, err
	}
	datastore.BoltDB = db

	// Device
	deviceDB, err := devicebuiltin.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.DeviceStore = deviceDB
	datastore.DeviceWorkerStore = deviceDB

	// User
	userDB, err := userbuiltin.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.UserStore = userDB
	datastore.UserWorkerStore = userDB

	// Profile
	profileDB, err := profilebuiltin.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.ProfileStore = profileDB

	// Blueprint
	blueprintDB, err := blueprintbuiltin.NewDB(db, profileDB)
	if err != nil {
		stdlog.Fatal(err)
	}
	datastore.BlueprintStore = blueprintDB
	datastore.BlueprintWorkerStore = blueprintDB

	// SCEP Challenge
	scepChallengeStore, err := challengestore.NewBoltDepot(db)
	if err != nil {
		return nil, err
	}
	datastore.SCEPChallengeStore = scepChallengeStore

	// DEP Sync
	depSyncDB, err := syncbuiltin.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.DEPSyncStore = depSyncDB
	datastore.DEPSyncWatcherStore = depSyncDB

	// Config
	configDB, err := configbuiltin.NewDB(db, pubClient)
	if err != nil {
		return nil, err
	}
	datastore.ConfigStore = configDB

	// Push Notification, APNS
	apnsStore, err := apnsbuiltin.NewDB(db, pubClient)
	if err != nil {
		return nil, err
	}
	datastore.APNSStore = apnsStore
	datastore.APNSWorkerStore = apnsStore

	// Remove
	removeDB, err := blockbuiltin.NewDB(db)
	datastore.RemoveStore = removeDB
	if err != nil {
		return nil, err
	}

	// SCEP
	scepDB, err := scepbuiltin.NewDB(db)
	datastore.RemoveStore = removeDB
	if err != nil {
		return nil, err
	}
	datastore.SCEPStore = scepDB

	return &datastore, nil
}

func createDB(configMap map[string]string) (*bolt.DB, error) {
	configPath := configMap["ConfigPath"]

	dbPath := filepath.Join(configPath, "micromdm.db")
	db, err := bolt.Open(dbPath, 0644, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "opening boltdb")
	}
	return db, nil
}
