package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	db, err := sql.Open("sqlite", "file:data/scheduler.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	container := sqlstore.NewWithDB(db, "sqlite3", waLog.Stdout("DB", "INFO", true))
	if err := container.Upgrade(ctx); err != nil {
		panic(err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(func(evt any) {
		switch e := evt.(type) {
		case *events.Connected:
			fmt.Println("EVENT connected")
		case *events.Disconnected:
			fmt.Println("EVENT disconnected")
		case *events.KeepAliveTimeout:
			fmt.Println("EVENT keepalive timeout", e.ErrorCount)
		case *events.KeepAliveRestored:
			fmt.Println("EVENT keepalive restored")
		case *events.StreamReplaced:
			fmt.Println("EVENT stream replaced")
		case *events.LoggedOut:
			fmt.Println("EVENT logged out", e.Reason)
		case *events.PairSuccess:
			fmt.Println("EVENT paired", e.ID)
		case *events.TemporaryBan:
			fmt.Println("EVENT temporary ban", e.Code, e.Expire)
		case *events.ClientOutdated:
			fmt.Println("EVENT client outdated")
		}
	})

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("QR", evt.Event)
			}
		}
	} else {
		if err := client.Connect(); err != nil {
			panic(err)
		}
	}

	fmt.Println("commands: state | contacts | send <number|me> <text> | quit")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "state":
			fmt.Println("connected:", client.IsConnected(), "loggedIn:", client.IsLoggedIn())
		case "contacts":
			contacts, err := client.Store.Contacts.GetAllContacts(ctx)
			if err != nil {
				fmt.Println("ERR", err)
				continue
			}
			fmt.Println("count:", len(contacts))
			n := 0
			for jid, c := range contacts {
				fmt.Printf("%s  full=%q push=%q\n", jid, c.FullName, c.PushName)
				if n++; n == 10 {
					break
				}
			}
		case "send":
			if len(args) < 3 {
				fmt.Println("usage: send <number|me> <text>")
				continue
			}
			var jid types.JID
			if args[1] == "me" {
				jid = client.Store.ID.ToNonAD()
			} else {
				jid = types.NewJID(args[1], types.DefaultUserServer)
			}
			text := strings.Join(args[2:], " ")
			resp, err := client.SendMessage(ctx, jid, &waE2E.Message{Conversation: proto.String(text)})
			if err != nil {
				fmt.Println("SEND ERR", err)
				continue
			}
			fmt.Println("sent id", resp.ID, "at", resp.Timestamp)
		case "quit":
			client.Disconnect()
			return
		}
	}
}
