package service

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm"
	"github.com/micromdm/micromdm/platform/command"
	"github.com/micromdm/micromdm/platform/pubsub"
	"github.com/micromdm/micromdm/platform/queue"
	"github.com/pkg/errors"
	"time"
)

type Queue interface {
	Next(ctx context.Context, resp mdm.Response) ([]byte, error)
}

type Store interface {
	Save(ctx context.Context, cmd *queue.DeviceCommand) error
	DeviceCommand(ctx context.Context, udid string) (*queue.DeviceCommand, error)
	UpdateCommandStatus(ctx context.Context, resp mdm.Response) error
}

type QueueService struct {
	Store          Store
	Logger         log.Logger
	withoutHistory bool
}

type Option func(*QueueService)

func WithLogger(logger log.Logger) Option {
	return func(s *QueueService) {
		s.Logger = logger
	}
}

func WithoutHistory() Option {
	return func(s *QueueService) {
		s.withoutHistory = true
	}
}

func (svc *QueueService) Next(ctx context.Context, resp mdm.Response) ([]byte, error) {
	cmd, err := svc.NextCommand(ctx, resp)
	if err != nil {
		return nil, err
	}
	if cmd == nil {
		return nil, nil
	}
	return cmd.Payload, nil
}

func (svc *QueueService) NextCommand(ctx context.Context, resp mdm.Response) (*queue.Command, error) {
	// The UDID is the primary key for the queue.
	// Depending on the enrollment type, replace the UDID with a different ID type.
	// UserID for managed user channel
	// EnrollmentID for BYOD User Enrollment.
	udid := resp.UDID
	if resp.UserID != nil {
		udid = *resp.UserID
	}
	if resp.EnrollmentID != nil {
		udid = *resp.EnrollmentID
	}

	err := svc.Store.UpdateCommandStatus(ctx, resp)
	if err != nil {
		return nil, err
	}

	dc, err := svc.Store.DeviceCommand(ctx, udid)
	if err != nil {
		if queue.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get device command from queue, udid: %s", resp.UDID)
	}

	var cmd *queue.Command
	switch resp.Status {
	case "NotNow":
		// We will try this command later when the device is not
		// responding with NotNow
		x, a := cut(dc.Commands, resp.CommandUUID)
		dc.Commands = a
		if x == nil {
			break
		}
		dc.NotNow = append(dc.NotNow, *x)

	case "Acknowledged":
		// move to completed, send next
		x, a := cut(dc.Commands, resp.CommandUUID)
		dc.Commands = a
		if x == nil {
			break
		}
		if !svc.withoutHistory {
			x.Acknowledged = time.Now().UTC()
			dc.Completed = append(dc.Completed, *x)
		}

	case "Error":
		// move to failed, send next
		x, a := cut(dc.Commands, resp.CommandUUID)

		dc.Commands = a
		if x == nil { // must've already bin ackd
			break
		}
		if !svc.withoutHistory {
			dc.Failed = append(dc.Failed, *x)
		}

	case "CommandFormatError":
		// move to failed
		x, a := cut(dc.Commands, resp.CommandUUID)
		dc.Commands = a
		if x == nil {
			break
		}
		if !svc.withoutHistory {
			dc.Failed = append(dc.Failed, *x)
		}

	case "Idle":
		// will send next command below

	default:
		return nil, fmt.Errorf("unknown response status: %s", resp.Status)
	}

	// pop the first command from the queue and add it to the end.
	// If the regular queue is empty, send a command that got
	// refused with NotNow before.
	cmd, dc.Commands = popFirst(dc.Commands)
	if cmd != nil {
		dc.Commands = append(dc.Commands, *cmd)
	} else if resp.Status != "NotNow" {
		cmd, dc.NotNow = popFirst(dc.NotNow)
		if cmd != nil {
			dc.Commands = append(dc.Commands, *cmd)
		}
	}

	if err := svc.Store.Save(ctx, dc); err != nil {
		return nil, err
	}
	return cmd, nil
}

func popFirst(all []queue.Command) (*queue.Command, []queue.Command) {
	if len(all) == 0 {
		return nil, all
	}
	first := all[0]
	all = append(all[:0], all[1:]...)
	return &first, all
}

func cut(all []queue.Command, uuid string) (*queue.Command, []queue.Command) {
	for i, cmd := range all {
		if cmd.UUID == uuid {
			all = append(all[:i], all[i+1:]...)
			return &cmd, all
		}
	}
	return nil, all
}

func NewQueue(store *Store, pubsub pubsub.PublishSubscriber, opts ...Option) (*QueueService, error) {
	svc := &QueueService{Store: *store, Logger: log.NewNopLogger()}
	for _, fn := range opts {
		fn(svc)
	}

	if err := svc.pollCommands(context.Background(), pubsub); err != nil {
		return nil, err
	}

	return svc, nil
}

func (svc *QueueService) pollCommands(ctx context.Context, pubsub pubsub.PublishSubscriber) error {
	commandEvents, err := pubsub.Subscribe(context.TODO(), "command-queue", command.CommandTopic)
	if err != nil {
		return errors.Wrapf(err,
			"subscribing push to %s topic", command.CommandTopic)
	}
	go func() {
		for {
			select {
			case event := <-commandEvents:
				var ev command.Event
				if err := command.UnmarshalEvent(event.Message, &ev); err != nil {
					level.Info(svc.Logger).Log("msg", "unmarshal command event in queue", "err", err)
					continue
				}

				cmd := new(queue.DeviceCommand)
				cmd.DeviceUDID = ev.DeviceUDID
				byUDID, err := svc.Store.DeviceCommand(ctx, ev.DeviceUDID)
				if err == nil && byUDID != nil {
					cmd = byUDID
				}
				newPayload, err := plist.Marshal(ev.Payload)
				if err != nil {
					level.Info(svc.Logger).Log("msg", "marshal event payload", "err", err)
					continue
				}
				newCmd := queue.Command{
					UUID:    ev.Payload.CommandUUID,
					Payload: newPayload,
				}
				cmd.Commands = append(cmd.Commands, newCmd)
				if err := svc.Store.Save(ctx, cmd); err != nil {
					level.Info(svc.Logger).Log("msg", "save command in db", "err", err)
					continue
				}
				level.Info(svc.Logger).Log(
					"msg", "queued event for device",
					"device_udid", ev.DeviceUDID,
					"command_uuid", ev.Payload.CommandUUID,
					"request_type", ev.Payload.Command.RequestType,
				)

				cq := new(queue.QueueCommandQueued)
				cq.DeviceUDID = ev.DeviceUDID
				cq.CommandUUID = ev.Payload.CommandUUID

				msgBytes, err := queue.MarshalQueuedCommand(cq)
				if err != nil {
					level.Info(svc.Logger).Log("msg", "marshal queued command", "err", err)
					continue
				}

				if err := pubsub.Publish(context.TODO(), queue.CommandQueuedTopic, msgBytes); err != nil {
					level.Info(svc.Logger).Log("msg", "publish command to queued topic", "err", err)
				}
			}
		}
	}()

	return nil
}
