package schema

import (
	"github.com/boltdb/bolt"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/micromdm/platform/apns"
	"github.com/micromdm/micromdm/platform/blueprint"
	"github.com/micromdm/micromdm/platform/config"
	"github.com/micromdm/micromdm/platform/dep/sync"
	"github.com/micromdm/micromdm/platform/device"
	"github.com/micromdm/micromdm/platform/profile"
	block "github.com/micromdm/micromdm/platform/remove"
	scepstore "github.com/micromdm/micromdm/platform/scep"
	"github.com/micromdm/micromdm/platform/user"
	"github.com/micromdm/scep/challenge"
)

type ImplementationID int

const (
	Bolt ImplementationID = iota
	Mysql
)

type Schema struct {
	ID                   ImplementationID
	BoltDB               *bolt.DB
	MysqlDB              *sqlx.DB
	APNSStore            apns.Store
	APNSWorkerStore      apns.WorkerStore
	BlueprintStore       blueprint.Store
	BlueprintWorkerStore blueprint.WorkerStore
	ConfigStore          config.Store
	DEPSyncStore         sync.Store
	DEPSyncWatcherStore  sync.WatcherStore
	DeviceStore          device.Store
	DeviceWorkerStore    device.WorkerStore
	ProfileStore         profile.Store
	RemoveStore          block.Store
	SCEPChallengeStore   challenge.Store
	SCEPStore            scepstore.Store
	UDIDCertAuthStore    device.UDIDCertAuthStore
	UserStore            user.Store
	UserWorkerStore      user.WorkerStore
}
