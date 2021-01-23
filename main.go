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

const (
	QrcodeLocation              = "appdata/qrcode.png"
	SwitchToSignalVideoLocation = "appdata/switch_to_signal_video.mp4"
	VideoSentJsonLocation       = "appdata/video_sent.json"
	SessionJsonLocation         = "appdata/session.json"
	ReplyTextLocation           = "appdata/reply_text.txt"
)

var (
	startTime = time.Now()
	videoSent = make(map[string]bool)
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

func isGroupMessage(remoteJid string) bool {
	return strings.Contains(remoteJid, "-")
}

func isStatusMessage(remoteJid string) bool {
	return strings.Contains(remoteJid, "status")
}

func sendSwitchedMessage(whatsappConn *whatsapp.Conn, remoteJid string) {
	_, err := whatsappConn.Send(whatsapp.TextMessage{
		Info: whatsapp.MessageInfo{
			RemoteJid: remoteJid,
		},
		Text: getReplyText(),
	})
	if err != nil {
		log.Printf("error sending message: %v", err)
	}
}

func getReplyText() string {
	content, err := ioutil.ReadFile(ReplyTextLocation)
	if err != nil {
		log.Println(err)
		return "*This user has switched to Signal.*\nKindly use Signal app to contact them."
	}
	// Convert []byte to string and print to screen
	return string(content)
}

func isFirstMessageFromContact(remoteJid string) bool {
	if videoSent[remoteJid] {
		return false
	} else {
		return true
	}
}

func sendSwitchVideo(whatsappConn *whatsapp.Conn, remoteJid string) {
	switchToSignalVideo, err := os.Open(SwitchToSignalVideoLocation)
	if err != nil {
		log.Printf("cannot find switch to signal video")
		return
	}
	defer switchToSignalVideo.Close()
	whatsappConn.Send(whatsapp.VideoMessage{
		Info: whatsapp.MessageInfo{
			RemoteJid: remoteJid,
		},
		Type:    "video/mp4",
		Caption: "Switch to Signal!",
		Content: switchToSignalVideo,
	})

}

func handleFirstMessageWithContact(whatsappConn *whatsapp.Conn, remoteJid string) {
	sendSwitchVideo(whatsappConn, remoteJid)
	videoSent[remoteJid] = true
	if err := serializeVideoSentMap(); err != nil {
		log.Printf("error in serialzing: %v\n", err)
	}
}

//Optional to be implemented. Implement HandleXXXMessage for the types you need.
func (h *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	if !message.Info.FromMe && startTime.Before(time.Unix(int64(message.Info.Timestamp), 0)) {
		remoteJid := message.Info.RemoteJid
		if !isGroupMessage(remoteJid) && !isStatusMessage(remoteJid) {
			sendSwitchedMessage(h.c, remoteJid)
			if isFirstMessageFromContact(remoteJid) {
				handleFirstMessageWithContact(h.c, remoteJid)
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

	err = deserializeVideoSentMap()
	if err != nil {
		log.Printf("error in deserialzing: %v\n", err)
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
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
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
			f, _ := os.Create(QrcodeLocation)
			_, _ = f.Write(png)
			fmt.Println("Qrcode Generated. Kindly scan and login")
			f.Close()
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

func deserializeVideoSentMap() error {
	if _, err := os.Stat(VideoSentJsonLocation); os.IsNotExist(err) {
		return serializeVideoSentMap()
	}
	JSONRaw, err := ioutil.ReadFile(VideoSentJsonLocation)
	err = json.Unmarshal(JSONRaw, &videoSent)
	return err
}

func serializeVideoSentMap() error {
	videoSentJSON, _ := json.Marshal(videoSent)
	err := ioutil.WriteFile(VideoSentJsonLocation, videoSentJSON, 0644)
	return err
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	JSONRaw, err := ioutil.ReadFile(SessionJsonLocation)
	err = json.Unmarshal(JSONRaw, &session)
	return session, err
}

func writeSession(session whatsapp.Session) error {
	sessionJSON, _ := json.Marshal(session)
	err := ioutil.WriteFile(SessionJsonLocation, sessionJSON, 0644)
	return err
}
