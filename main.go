package main

import (
	"encoding/json"
	"fmt"
	"github.com/skip2/go-qrcode"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Rhymen/go-whatsapp"
)

var (
	startTime = time.Now()
)

type waHandler struct {
	c *whatsapp.Conn
}

//HandleError needs to be implemented to be a valid WhatsApp handler
func (h *waHandler) HandleError(err error) {

	if e, ok := err.(*whatsapp.ErrConnectionFailed); ok {
		log.Printf("Connection failed, underlying error: %v", e.Err)
		log.Println("Waiting 30sec...")
		<-time.After(30 * time.Second)
		log.Println("Reconnecting...")
		err := h.c.Restore()
		if err != nil {
			log.Fatalf("Restore failed: %v", err)
		}
	} else {
		log.Printf("error occoured: %v\n", err)
	}
}

func isGroupMessage(message whatsapp.TextMessage) bool {
	return strings.Contains(message.Info.RemoteJid, "-")
}

//Optional to be implemented. Implement HandleXXXMessage for the types you need.
func (h *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	if !message.Info.FromMe && startTime.Before(time.Unix(int64(message.Info.Timestamp), 0)) {
		if !isGroupMessage(message) {
			_, err := h.c.Send(whatsapp.TextMessage{
				Info: whatsapp.MessageInfo{
					RemoteJid: message.Info.RemoteJid,
				},
				Text: "*This User has switched to Signal.*\nKindly use Signal app to contact him.",
			})
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error sending message: %v", err)
				return
			}
		}
	}
}

func main() {
	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		log.Fatalf("error creating connection: %v\n", err)
	}
	err = wac.SetClientName("Google Chrome", "Chrome", "87")
	if err != nil {
		log.Fatalf("error setting client name: %v\n", err)
	}

	//Add handler
	wac.AddHandler(&waHandler{wac})

	//login or restore
	if err := login(wac); err != nil {
		log.Fatalf("error logging in: %v\n", err)
	}

	//verifies phone connectivity
	pong, err := wac.AdminTest()

	if !pong || err != nil {
		log.Fatalf("error pinging in: %v\n", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	//Disconnect safe
	fmt.Println("Shutting down now.")
	session, err := wac.Disconnect()
	if err != nil {
		log.Fatalf("error disconnecting: %v\n", err)
	}
	if err := writeSession(session); err != nil {
		log.Fatalf("error saving session: %v", err)
	}
}

func login(wac *whatsapp.Conn) error {
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreWithSession(session)
		if err != nil {
			return fmt.Errorf("restoring failed: %v\n", err)
		}
	} else {
		//no saved session -> regular login
		qr := make(chan string)
		go func() {
			var png []byte
			png, _ = qrcode.Encode(<-qr, qrcode.Highest, 512)
			f, _ := os.Create("qrcode.png")
			_, _ = f.Write(png)
			fmt.Println("Qrcode Generated. Kindly scan and login")
		}()
		session, err = wac.Login(qr)
		if err != nil {
			return fmt.Errorf("error during login: %v\n", err)
		}
	}
	//save session
	err = writeSession(session)
	if err != nil {
		return fmt.Errorf("error saving session: %v\n", err)
	}
	return nil
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	JSONRaw, err := ioutil.ReadFile("session.json")
	err = json.Unmarshal(JSONRaw, &session)
	return session, err
}

func writeSession(session whatsapp.Session) error {
	sessionJSON, _ := json.Marshal(session)
	err := ioutil.WriteFile("session.json", sessionJSON, 0644)
	return err
}
