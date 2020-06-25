package builtin

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/go-kit/kit/log"
	"github.com/micromdm/micromdm/platform/queue"
	"github.com/micromdm/micromdm/mdm"
)

func TestNext_Error(t *testing.T) {
	store, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, queue.Command{UUID: "xCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "yCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "zCmd"})
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	resp := mdm.Response{
		UDID:        dc.DeviceUDID,
		CommandUUID: "xCmd",
		Status:      "Error",
	}
	for range dc.Commands {
		cmd, err := store.nextCommand(ctx, resp)
		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}
		if cmd == nil {
			t.Fatal("expected cmd but got nil")
		}

		if have, errd := cmd.UUID, resp.CommandUUID; have == errd {
			t.Error("got back command which previously failed")
		}
	}
}

func TestNext_NotNow(t *testing.T) {
	store, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, queue.Command{UUID: "xCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "yCmd"})
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	tf := func(t *testing.T) {

		resp := mdm.Response{
			UDID:        dc.DeviceUDID,
			CommandUUID: "yCmd",
			Status:      "NotNow",
		}
		cmd, err := store.nextCommand(ctx, resp)

		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}

		resp = mdm.Response{
			UDID:        dc.DeviceUDID,
			CommandUUID: cmd.UUID,
			Status:      "NotNow",
		}

		cmd, err = store.nextCommand(ctx, resp)
		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}
		if cmd != nil {
			t.Error("Got back a notnowed command.")
		}
	}

	t.Run("withManyCommands", tf)
	dc.Commands = []queue.Command{{UUID: "xCmd"}}
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}
	t.Run("withOneCommand", tf)
}

func TestNext_Idle(t *testing.T) {
	store, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	dc.Commands = append(dc.Commands, queue.Command{UUID: "xCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "yCmd"})
	dc.Commands = append(dc.Commands, queue.Command{UUID: "zCmd"})
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	resp := mdm.Response{
		UDID:        dc.DeviceUDID,
		CommandUUID: "xCmd",
		Status:      "Idle",
	}
	for i := range dc.Commands {
		cmd, err := store.nextCommand(ctx, resp)
		if err != nil {
			t.Fatalf("expected nil, but got err: %s", err)
		}
		if cmd == nil {
			t.Fatal("expected cmd but got nil")
		}

		if have, want := cmd.UUID, dc.Commands[i].UUID; have != want {
			t.Errorf("have %s, want %s, index %d", have, want, i)
		}
	}
}

func TestNext_zeroCommands(t *testing.T) {
	store, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()

	dc := &queue.DeviceCommand{DeviceUDID: "TestDevice"}
	if err := store.Save(ctx, dc); err != nil {
		t.Fatal(err)
	}

	var allStatuses = []string{
		"Acknowledged",
		"NotNow",
	}

	for _, s := range allStatuses {
		t.Run(s, func(t *testing.T) {
			resp := mdm.Response{CommandUUID: s, Status: s}
			cmd, err := store.nextCommand(ctx, resp)
			if err != nil {
				t.Errorf("expected nil, but got err: %s", err)
			}
			if cmd != nil {
				t.Errorf("expected nil cmd but got %s", cmd.UUID)
			}
		})
	}

}

func setupDB(t *testing.T) (*DBWrapper, func()) {
	f, _ := ioutil.TempFile("", "bolt-")
	teardown := func() {
		f.Close()
		os.Remove(f.Name())
	}

	db, err := bolt.Open(f.Name(), 0777, nil)
	if err != nil {
		t.Fatalf("couldn't open bolt, err %s\n", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(queue.DeviceCommandBucket))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &DBWrapper{DB: db, logger: log.NewNopLogger()}
	return store, teardown
}
