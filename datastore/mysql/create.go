package mysql

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/kolide/kit/dbutil"
	"github.com/micromdm/micromdm/datastore/schema"
	apnsmysql "github.com/micromdm/micromdm/platform/apns/mysql"
	blueprintbuiltin "github.com/micromdm/micromdm/platform/blueprint/builtin"
	configmysql "github.com/micromdm/micromdm/platform/config/mysql"
	syncmysql "github.com/micromdm/micromdm/platform/dep/sync/mysql"
	devicemysql "github.com/micromdm/micromdm/platform/device/mysql"
	profilemysql "github.com/micromdm/micromdm/platform/profile/mysql"
	"github.com/micromdm/micromdm/platform/pubsub"
	queueMysql "github.com/micromdm/micromdm/platform/queue/mysql"
	blockmysql "github.com/micromdm/micromdm/platform/remove/mysql"
	scepmysql "github.com/micromdm/micromdm/platform/scep/mysql"
	userbuiltin "github.com/micromdm/micromdm/platform/user/builtin"
	challengestore "github.com/micromdm/scep/challenge/bolt"
	"github.com/pkg/errors"
	stdlog "log"
	"path/filepath"
	"time"
)

func Create(configMap map[string]string, pubClient pubsub.PublishSubscriber) (*schema.Schema, error) {
	datastore := schema.Schema{
		ID: schema.Mysql,
	}

	// MySQL Database
	db, err := createDB(configMap)
	if err != nil {
		return nil, err
	}

	// Bolt Database used until all entities are implemented in MySQL
	boltDB, err := createBoltDB(configMap)
	if err != nil {
		return nil, err
	}

	// Device
	deviceDB, err := devicemysql.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.DeviceStore = deviceDB
	datastore.DeviceWorkerStore = deviceDB
	datastore.UDIDCertAuthStore = deviceDB

	// User
	userDB, err := userbuiltin.NewDB(boltDB)
	if err != nil {
		return nil, err
	}
	datastore.UserStore = userDB
	datastore.UserWorkerStore = userDB
	// TODO: implement UserStore and UserWorkerStore in MySQL
	fmt.Println("warning", "TODO: implement UserStore and UserWorkerStore in MySQL")

	// Profile
	profileDB, err := profilemysql.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.ProfileStore = profileDB

	// Blueprint
	blueprintDB, err := blueprintbuiltin.NewDB(boltDB, profileDB)
	if err != nil {
		stdlog.Fatal(err)
	}
	datastore.BlueprintStore = blueprintDB
	datastore.BlueprintWorkerStore = blueprintDB
	//TODO: implement BlueprintStore and BlueprintWorkerStore in MySQL
	fmt.Println("warning", "TODO: implement BlueprintStore and BlueprintWorkerStore in MySQL")

	// SCEP Challenge
	scepChallengeStore, err := challengestore.NewBoltDepot(boltDB)
	if err != nil {
		return nil, err
	}
	datastore.SCEPChallengeStore = scepChallengeStore
	// TODO: implement SCEPChallengeStore in MySQL
	fmt.Println("warning", "TODO: implement SCEPChallengeStore in MySQL")

	// DEP Sync
	depSyncDB, err := syncmysql.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.DEPSyncStore = depSyncDB
	datastore.DEPSyncWatcherStore = depSyncDB

	// Config
	configDB, err := configmysql.NewDB(db, pubClient)
	if err != nil {
		return nil, err
	}
	datastore.ConfigStore = configDB

	// Push Notification, APNS
	apnsStore, err := apnsmysql.NewDB(db, pubClient)
	if err != nil {
		return nil, err
	}
	datastore.APNSStore = apnsStore
	datastore.APNSWorkerStore = apnsStore

	// Remove
	removeDB, err := blockmysql.NewDB(db)
	datastore.RemoveStore = removeDB
	if err != nil {
		return nil, err
	}

	// SCEP
	scepDB, err := scepmysql.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.SCEPStore = scepDB

	// Queue
	queueDB, err := queueMysql.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.QueueStore = queueDB

	return &datastore, nil
}

func createDB(configMap map[string]string) (*sqlx.DB, error) {
	mysqlUsername := configMap["MysqlUsername"]
	mysqlPassword := configMap["MysqlPassword"]
	mysqlDatabase := configMap["MysqlDatabase"]
	mysqlHost := configMap["MysqlHost"]
	mysqlPort := configMap["MysqlPort"]

	if mysqlUsername == "" {
		return nil, errors.New("missing MySQL username")
	}

	if mysqlPassword == "" {
		return nil, errors.New("missing MySQL password")
	}

	if mysqlHost == "" {
		return nil, errors.New("missing MySQL host")
	}

	if mysqlPort == "" {
		return nil, errors.New("missing MySQL port")
	}

	if mysqlDatabase == "" {
		return nil, errors.New("missing MySQL database name")
	}

	db, err := dbutil.OpenDBX(
		"mysql",
		mysqlUsername+":"+mysqlPassword+"@tcp("+mysqlHost+":"+mysqlPort+")/"+mysqlDatabase+"?parseTime=true",
		dbutil.WithLogger(log.NewNopLogger()),
		dbutil.WithMaxAttempts(1),
	)
	if err != nil {
		return nil, errors.Wrap(err, "opening mysql")
	}

	// Set the number of open and idle connection to a maximum total of 2.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

func createBoltDB(configMap map[string]string) (*bolt.DB, error) {
	configPath := configMap["ConfigPath"]

	dbPath := filepath.Join(configPath, "micromdm_mysql_not_yet_implemented.db")
	db, err := bolt.Open(dbPath, 0644, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "opening boltdb for supporting MySQL entities not yet implemented")
	}
	return db, nil
}
