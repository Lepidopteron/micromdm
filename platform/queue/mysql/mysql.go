package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	sq "gopkg.in/Masterminds/squirrel.v1"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/micromdm/micromdm/mdm"
	"github.com/micromdm/micromdm/platform/queue"
)

const (
	DeviceCommandTable = "device_commands"
)

type DB struct {
	*sqlx.DB
}

type MysqlCommand struct {
	UUID    string			`db:"uuid"`
	DeviceUDID string		`db:"device_udid"`
	Payload []byte			`db:"payload"`

	CreatedAt    time.Time 	`db:"created_at"`
	LastSentAt   time.Time 	`db:"last_sent_at"`
	Acknowledged time.Time 	`db:"acknowledged_at"`

	TimesSent int 			`db:"times_sent"`

	LastStatus     string 	`db:"last_status"`
	FailureMessage []byte 	`db:"failure_message"`
	Order			int		`db:"command_order"`
}

func command_columns() []string {
	return []string{
		"uuid",
		"device_udid",
		"payload",
		"created_at",
		"last_sent_at",
		"acknowledged_at",
		"times_sent",
		"last_status",
		"failure_message",
		"command_order",
	}
}

func SetupDB(db *sqlx.DB) error {
	// Required for TIMESTAMP DEFAULT 0
	_,err := db.Exec(`SET sql_mode = '';`)
	if err != nil {
		return errors.Wrap(err, "setting sql_mode")
	}

	// "github.com/micromdm/micromdm/platform/queue/internal/devicecommandproto"
	_,err = db.Exec(`CREATE TABLE IF NOT EXISTS `+DeviceCommandTable+` (
	    uuid VARCHAR(40) PRIMARY KEY,
	    device_udid VARCHAR(40) NOT NULL,
	    payload BLOB DEFAULT NULL,
	    created_at TIMESTAMP DEFAULT 0,
	    last_sent_at TIMESTAMP DEFAULT 0,
	    acknowledged_at TIMESTAMP DEFAULT 0,
	    times_sent int(11) DEFAULT 0,
	    last_status VARCHAR(32) DEFAULT NULL,
	    failure_message BLOB DEFAULT NULL,
	    command_order int(11) DEFAULT 0
	);`)

	if err != nil {
	   return errors.Wrap(err, "creating "+DeviceCommandTable+" sql table failed")
	}
	
	_,err = db.Exec(`ALTER TABLE `+DeviceCommandTable+` MODIFY payload MEDIUMBLOB DEFAULT NULL;`)
	if err != nil {
	   return errors.Wrap(err, "altering "+DeviceCommandTable+" sql table failed")
	}

	return nil
}

func NewDB(db *sqlx.DB) (*DB, error) {
	err := SetupDB(db)
	if err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

func (db *DB) SaveCommand(ctx context.Context, cmd queue.Command, deviceUDID string, order int) error {
	// Make sure we take the time offset into account for "zero" dates	
	t := time.Now()
	_, offset := t.Zone()

	// Don't multiply by zero
	if offset <= 0 {
		offset = 1
	}
	var minTimestampSec int64 = int64(offset) * 60 * 60 * 24
	
	if cmd.CreatedAt.IsZero() || cmd.CreatedAt.Unix() < minTimestampSec {
		cmd.CreatedAt = time.Unix(minTimestampSec, 0)
	}
	
	if cmd.LastSentAt.IsZero() || cmd.LastSentAt.Unix() < minTimestampSec {
		cmd.LastSentAt = time.Unix(minTimestampSec, 0)
	}
	
	if cmd.Acknowledged.IsZero() || cmd.Acknowledged.Unix() < minTimestampSec {
		cmd.Acknowledged = time.Unix(minTimestampSec, 0)
	}
	
	updateQuery, argsUpdate, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Update(DeviceCommandTable).
		Prefix("ON DUPLICATE KEY").
		Set("uuid", cmd.UUID).
		Set("device_udid", deviceUDID).
		Set("payload", cmd.Payload).
		Set("created_at", cmd.CreatedAt).
		Set("last_sent_at", cmd.LastSentAt).
		Set("acknowledged_at", cmd.Acknowledged).
		Set("times_sent", cmd.TimesSent).
		Set("last_status", cmd.LastStatus).
		Set("failure_message", cmd.FailureMessage).
		Set("command_order", order).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building update query for command save")
	}
	
	// MySql Convention
	// Replace "ON DUPLICATE KEY UPDATE TABLE_NAME SET" to "ON DUPLICATE KEY UPDATE"
	updateQuery = strings.Replace(updateQuery, DeviceCommandTable+" SET ", "", -1)

	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Insert(DeviceCommandTable).
		Columns(command_columns()...).
		Values(
			cmd.UUID,
			deviceUDID,
			cmd.Payload,
			cmd.CreatedAt,
			cmd.LastSentAt,
			cmd.Acknowledged,
			cmd.TimesSent,
			cmd.LastStatus,
			cmd.FailureMessage,
			order,
		).
		Suffix(updateQuery).
		ToSql()
	
	var allArgs = append(args, argsUpdate...)
	
	if err != nil {
		return errors.Wrap(err, "building command save query")
	}
	
	_, err = db.DB.ExecContext(ctx, query, allArgs...)
	
	return errors.Wrap(err, "exec command save in mysql")
}

func (db *DB) Save(ctx context.Context, cmd *queue.DeviceCommand) error {
	//err := SetupDB(db.DB)
	//if err != nil {
	//	return err
	//}
	
	for i, _command := range cmd.Commands {
		err := db.SaveCommand(ctx, _command, cmd.DeviceUDID, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) DeviceCommand(ctx context.Context, udid string) (*queue.DeviceCommand, error) {
	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Select(command_columns()...).
		From(DeviceCommandTable).
		Where(sq.Eq{"device_udid": udid}).
		OrderBy("command_order").
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "building sql")
	}

	var list []MysqlCommand
	err = db.DB.SelectContext(ctx, &list, query, args...)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, &queue.NotFound{ResourceType: "DeviceCommand", Message: fmt.Sprintf("udid %s", udid)}
	}
	dev, _err := UnmarshalMysqlCommand(udid, list)
	if _err != nil {
		return nil, _err
	}
	return &dev, errors.Wrap(err, "finding device_commands by udid")
}

func (db *DB) UpdateCommandStatus(ctx context.Context, resp mdm.Response) error {
	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Question).
		Update(DeviceCommandTable).
		Set("last_status", resp.Status).
		Where(sq.Eq{"uuid": resp.CommandUUID}).
		ToSql()
	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "building update query for command save")
	}
	
	return errors.Wrap(err, "exec command save in mysql")
}

func UnmarshalMysqlCommand(udid string, mysqlCommands []MysqlCommand) (queue.DeviceCommand, error) {
	var dev = queue.DeviceCommand {
		DeviceUDID: udid,
	}
	
	for _, command := range mysqlCommands {
		if command.DeviceUDID == udid {
			var cmd = queue.Command {
				UUID:         	command.UUID,
				Payload:      	command.Payload,
				CreatedAt:    	command.CreatedAt,
				LastSentAt:   	command.LastSentAt,
				Acknowledged: 	command.Acknowledged,
	
				TimesSent: 		command.TimesSent,
	
				LastStatus:     command.LastStatus,
				FailureMessage: command.FailureMessage,
			}
			
			switch cmd.LastStatus {
			case "NotNow":
				dev.NotNow = append(dev.NotNow, cmd)
		
			case "Acknowledged":
				dev.Completed = append(dev.Completed, cmd)
				
			case "Error":
				dev.Failed = append(dev.Failed, cmd)
						
			case "CommandFormatError":
				dev.Failed = append(dev.Failed, cmd)
		
			case "Idle":
				// will send next command below
				dev.Commands = append(dev.Commands, cmd)
		
			default:
				// Not yet classified
				dev.Commands = append(dev.Commands, cmd)
			}
		}
	}
	return dev, nil
}