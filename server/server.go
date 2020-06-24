package server

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/micromdm/micromdm/datastore"
	"github.com/micromdm/micromdm/datastore/schema"
	scep "github.com/micromdm/scep/server"
	"github.com/pkg/errors"
	"net/http"
	"net/url"

	"github.com/micromdm/micromdm/dep"
	"github.com/micromdm/micromdm/mdm"
	"github.com/micromdm/micromdm/mdm/enroll"
	"github.com/micromdm/micromdm/platform/apns"
	"github.com/micromdm/micromdm/platform/command"
	"github.com/micromdm/micromdm/platform/config"
	"github.com/micromdm/micromdm/platform/dep/sync"
	"github.com/micromdm/micromdm/platform/device"
	devicebuiltin "github.com/micromdm/micromdm/platform/device/builtin"
	devicemysql "github.com/micromdm/micromdm/platform/device/mysql"
	"github.com/micromdm/micromdm/platform/pubsub"
	"github.com/micromdm/micromdm/platform/pubsub/inmem"

	"github.com/micromdm/micromdm/platform/queue"
	queueBuiltin "github.com/micromdm/micromdm/platform/queue/builtin"
	queueMysql "github.com/micromdm/micromdm/platform/queue/mysql"

	block "github.com/micromdm/micromdm/platform/remove"
	"github.com/micromdm/micromdm/workflow/webhook"
)

type Server struct {
	ConfigPath          string
	Depsim              string
	ServerPublicURL     string
	SCEPChallenge       string
	SCEPClientValidity  int
	TLSCertPath         string
	UseDynSCEPChallenge bool
	GenDynSCEPChallenge bool
	CommandWebhookURL   string
	NoCmdHistory        bool
	MysqlUsername       string
	MysqlPassword       string
	MysqlDatabase       string
	MysqlHost           string
	MysqlPort           string

	Datastore			*schema.Schema

	APNSPushService      apns.Service
	CommandService       command.Service
	ConfigService        config.Service
	EnrollService        enroll.Service
	MDMService           mdm.Service
	SCEPService          scep.Service

	DEPClient            *dep.Client
	PubClient            pubsub.PublishSubscriber
	WebhooksHTTPClient   *http.Client
}

func (c *Server) Setup(logger log.Logger) error {
	if err := c.setupPubSub(); err != nil {
		return err
	}

	configMap := map[string]string {
		"MysqlUsername": c.MysqlUsername,
		"MysqlPassword": c.MysqlPassword,
		"MysqlDatabase": c.MysqlDatabase,
		"MysqlHost": c.MysqlHost,
		"MysqlPort": c.MysqlPort,
		"ConfigPath": c.ConfigPath,
	}

	datastore, err := datastore.Create(c.getDatastoreID(), configMap, c.PubClient)
	if err != nil {
		return err
	}
	c.Datastore = datastore

	if err := c.setupSCEP(logger); err != nil {
		return err
	}

	if err := c.setupPushService(logger); err != nil {
		return err
	}

	if err := c.setupCommandService(); err != nil {
		return err
	}

	if err := c.setupWebhooks(logger); err != nil {
		return err
	}

	if err := c.setupCommandQueue(logger); err != nil {
		return err
	}

	if err := c.setupDepClient(); err != nil {
		return err
	}

	if err := c.setupEnrollmentService(); err != nil {
		return err
	}

	return nil
}

func (c *Server) getDatastoreID() schema.ImplementationID {
	// If Mysql is set up, use Mysql, else use Bolt as Fallback
	if c.MysqlUsername != "" && c.MysqlPassword != "" && c.MysqlHost != "" && c.MysqlPort != "" && c.MysqlDatabase != "" {
		return schema.MySQL
	}
	return schema.Bolt
}

func (c *Server) setupPubSub() error {
	c.PubClient = inmem.NewPubSub()
	return nil
}

func (c *Server) setupWebhooks(logger log.Logger) error {
	if c.CommandWebhookURL == "" {
		return nil
	}

	ctx := context.Background()
	ww := webhook.New(c.CommandWebhookURL, c.PubClient, webhook.WithLogger(logger), webhook.WithHTTPClient(c.WebhooksHTTPClient))
	go ww.Run(ctx)
	return nil
}

func (c *Server) setupCommandService() error {
	commandService, err := command.New(c.PubClient)
	if err != nil {
		return err
	}
	c.CommandService = commandService
	return nil
}

func (c *Server) setupCommandQueue(logger log.Logger) error {
	var q queue.Store
	var udidCertAuthStore device.UDIDCertAuthStore
	var err error

	if c.Datastore.ID == schema.MySQL {
		opts := []queueMysql.Option{queueMysql.WithLogger(logger)}
		if c.NoCmdHistory {
			opts = append(opts, queueMysql.WithoutHistory())
		}

		q, err = queueMysql.NewQueue(c.Datastore.MysqlDB, c.PubClient, opts...)
		if err != nil {
			return err
		}

		udidCertAuthStore, err = devicemysql.NewDB(c.Datastore.MysqlDB)
		if err != nil {
			return errors.Wrap(err, "new device db")
		}
	} else {
		opts := []queueBuiltin.Option{queueBuiltin.WithLogger(logger)}
		if c.NoCmdHistory {
			opts = append(opts, queueBuiltin.WithoutHistory())
		}
		
		q, err = queueBuiltin.NewQueue(c.Datastore.BoltDB, c.PubClient, opts...)
		if err != nil {
			return err
		}

		udidCertAuthStore, err = devicebuiltin.NewDB(c.Datastore.BoltDB)
		if err != nil {
			return errors.Wrap(err, "new device db")
		}
	}

	var mdmService mdm.Service
	{
		svc := mdm.NewService(c.PubClient, q)
		mdmService = svc
		mdmService = block.RemoveMiddleware(c.Datastore.RemoveStore)(mdmService)

		udidauthLogger := log.With(logger, "component", "udidcertauth")
		mdmService = device.UDIDCertAuthMiddleware(udidCertAuthStore, udidauthLogger)(mdmService)
	
		verifycertLogger := log.With(logger, "component", "verifycert")
		mdmService = VerifyCertificateMiddleware(c.Datastore.SCEPStore, verifycertLogger)(mdmService)
	}
	c.MDMService = mdmService

	return nil
}

func (c *Server) setupPushService(logger log.Logger) error {
	service, err := apns.New(c.Datastore.APNSStore, c.Datastore.ConfigStore, c.PubClient)
	if err != nil {
		return errors.Wrap(err, "starting micromdm push service")
	}

	c.APNSPushService = apns.LoggingMiddleware(
		log.With(level.Info(logger), "component", "apns"),
	)(service)

	worker := apns.NewWorker(c.Datastore.APNSWorkerStore, c.PubClient, logger)
	go worker.Run(context.Background())

	return nil
}

func (c *Server) setupEnrollmentService() error {
	var (
		SCEPCertificateSubject string
		err                    error
	)

	challengeStore := c.Datastore.SCEPChallengeStore
	if !c.GenDynSCEPChallenge {
		challengeStore = nil
	}

	// TODO: clean up order of inputs. Maybe pass *SCEPConfig as an arg?
	// but if you do, the packages are coupled, better not.
	c.EnrollService, err = enroll.NewService(
		c.Datastore.ConfigStore,
		c.PubClient,
		c.ServerPublicURL+"/scep",
		c.SCEPChallenge,
		c.ServerPublicURL,
		c.TLSCertPath,
		SCEPCertificateSubject,
		c.Datastore.ProfileStore,
		challengeStore,
	)
	return errors.Wrap(err, "setting up enrollment service")
}

func (c *Server) setupDepClient() error {
	var (
		conf           dep.OAuthParameters
		depsim         = c.Depsim
		hasTokenConfig bool
		opts           []dep.Option
	)

	// try getting the oauth config from bolt
	ctx := context.Background()
	tokens, err := c.Datastore.ConfigStore.DEPTokens(ctx)
	if err != nil {
		return err
	}
	if len(tokens) >= 1 {
		hasTokenConfig = true
		conf.ConsumerSecret = tokens[0].ConsumerSecret
		conf.ConsumerKey = tokens[0].ConsumerKey
		conf.AccessSecret = tokens[0].AccessSecret
		conf.AccessToken = tokens[0].AccessToken
		// TODO: handle expiration
	}

	// override with depsim keys if specified on CLI
	if depsim != "" {
		hasTokenConfig = true
		conf = dep.OAuthParameters{
			ConsumerKey:    "CK_48dd68d198350f51258e885ce9a5c37ab7f98543c4a697323d75682a6c10a32501cb247e3db08105db868f73f2c972bdb6ae77112aea803b9219eb52689d42e6",
			ConsumerSecret: "CS_34c7b2b531a600d99a0e4edcf4a78ded79b86ef318118c2f5bcfee1b011108c32d5302df801adbe29d446eb78f02b13144e323eb9aad51c79f01e50cb45c3a68",
			AccessToken:    "AT_927696831c59ba510cfe4ec1a69e5267c19881257d4bca2906a99d0785b785a6f6fdeb09774954fdd5e2d0ad952e3af52c6d8d2f21c924ba0caf4a031c158b89",
			AccessSecret:   "AS_c31afd7a09691d83548489336e8ff1cb11b82b6bca13f793344496a556b1f4972eaff4dde6deb5ac9cf076fdfa97ec97699c34d515947b9cf9ed31c99dded6ba",
		}
		depsimurl, err := url.Parse(depsim)
		if err != nil {
			return err
		}
		opts = append(opts, dep.WithServerURL(depsimurl))
	}

	if !hasTokenConfig {
		return nil
	}

	c.DEPClient = dep.NewClient(conf, opts...)
	return nil
}

func (c *Server) CreateDEPSyncer(logger log.Logger) (sync.Syncer, error) {
	client := c.DEPClient
	opts := []sync.Option{
		sync.WithLogger(log.With(logger, "component", "depsync")),
	}
	if client != nil {
		opts = append(opts, sync.WithClient(client))
	}

	var err error
	var syncer sync.Syncer
	syncer, err = sync.NewWatcher(c.Datastore.DEPSyncWatcherStore, c.PubClient, opts...)
	if err != nil {
		return nil, err
	}

	return syncer, nil
}

func (c *Server) setupSCEP(logger log.Logger) error {

	var err error
	
	opts := []scep.ServiceOption{
		scep.ClientValidity(c.SCEPClientValidity),
	}
	
	var scepChallengeOption scep.ServiceOption
	if c.UseDynSCEPChallenge {
		scepChallengeOption = scep.WithDynamicChallenges(c.Datastore.SCEPChallengeStore)
	} else {
		scepChallengeOption = scep.ChallengePassword(c.SCEPChallenge)
	}
	opts = append(opts, scepChallengeOption)

	key, err := c.Datastore.SCEPStore.CreateOrLoadKey(2048)
	if err != nil {
		return err
	}

	fmt.Println("CreateOrLoadCA")
	_, err = c.Datastore.SCEPStore.CreateOrLoadCA(key, 5, "MicroMDM", "US")
	if err != nil {
		return err
	}

	c.SCEPService, err = scep.NewService(c.Datastore.SCEPStore, opts...)
	if err != nil {
		return err
	}

	c.SCEPService = scep.NewLoggingService(logger, c.SCEPService)

	return nil
}
