package mysql

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/kolide/kit/dbutil"
	"github.com/micromdm/micromdm/datastore/schema"
	apnsmysql "github.com/micromdm/micromdm/platform/apns/mysql"
	configmysql "github.com/micromdm/micromdm/platform/config/mysql"
	syncmysql "github.com/micromdm/micromdm/platform/dep/sync/mysql"
	devicemysql "github.com/micromdm/micromdm/platform/device/mysql"
	profilemysql "github.com/micromdm/micromdm/platform/profile/mysql"
	"github.com/micromdm/micromdm/platform/pubsub"
	queueMysql "github.com/micromdm/micromdm/platform/queue/mysql"
	blockmysql "github.com/micromdm/micromdm/platform/remove/mysql"
	scepmysql "github.com/micromdm/micromdm/platform/scep/mysql"
	"github.com/pkg/errors"
)

func Create(configMap map[string]string, pubClient pubsub.PublishSubscriber) (*schema.Schema, error) {
	datastore := schema.Schema{
		ID: schema.Mysql,
	}

	// Database
	db, err := createDB(configMap)
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
	//userDB, err := devicemysql.NewDB(db)
	//if err != nil {
	//	return nil, err
	//}
	//datastore.UserStore = userDB
	//datastore.UserWorkerStore = userDB
	// TODO: implement MySQL UserStore and UserWorkerStore
	fmt.Println("warning", "TODO: implement MySQL UserStore and UserWorkerStore")

	// Profile
	profileDB, err := profilemysql.NewDB(db)
	if err != nil {
		return nil, err
	}
	datastore.ProfileStore = profileDB

	// Blueprint
	//blueprintDB, err := blueprintmysql.NewDB(db, datastore.ProfileStore)
	//if err != nil {
	//	stdlog.Fatal(err)
	//}
	//datastore.BlueprintStore = blueprintDB
	//datastore.BlueprintWorkerStore = blueprintDB
	// TODO: implement MySQL BlueprintStore and BlueprintWorkerStore
	fmt.Println("warning", "TODO: implement MySQL BlueprintStore and BlueprintWorkerStore")

	// SCEP Challenge
	//scepChallengeStore, err := ???.???(db)
	//if err != nil {
	//	return nil, err
	//}
	//datastore.SCEPChallengeStore = scepChallengeStore
	// TODO: implement MySQL UserStore and UserWorkerStore
	fmt.Println("warning", "TODO: implement MySQL SCEPChallengeStore)")

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
	queueDB, err :=  queueMysql.NewDB(db)
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
