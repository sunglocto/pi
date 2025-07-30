package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
	"log"
	"os"
	"time"
)

var client xmpp.Sender

type Account struct {
	jid         string
	password    string
	server      string
	defaultNick string
	Contacts    []Chat
}

type Chat struct {
	jid   string
	isMuc bool
	nick  string
}

type ActiveChat struct {
	Chat    Chat
	Account Account
}

var ActiveChats []ActiveChat
var Accounts []Account

// TODO: LOAD ACCOUNTS FROM FILESYSTEM
func main() {

	//createAccountGUI()
	myAccount := Account{}
	myAccount.jid = "moron@example.com"
	myAccount.password = "123456789"
	myAccount.defaultNick = "moron"
	myAccount.server = "example.com:5222"
	Accounts = append(Accounts, myAccount)
	a := app.New()
	w := a.NewWindow("pi")
	w.Resize(fyne.NewSize(500, 400))
	acc := Accounts[0]
	NewChat := Chat{jid: "someguysmuc@muc.example.com", isMuc: true, nick: myAccount.defaultNick}
	acc.Contacts = append(acc.Contacts, NewChat)
	ActiveChats = append(ActiveChats, ActiveChat{NewChat, Accounts[0]})

	// Chat area
	chatLog := widget.NewMultiLineEntry()
	chatLog.Wrapping = fyne.TextWrapWord
	chatLog.Disable()

	// Message entry
	msgEntry := widget.NewEntry()
	msgEntry.SetPlaceHolder("Type a message...")

	// Send button
	sendBtn := widget.NewButton("Send", func() {
		if client != nil && msgEntry.Text != "" {
			var typ stanza.StanzaType
			if ActiveChats[0].Chat.isMuc {
				typ = stanza.MessageTypeGroupchat
			} else {
				typ = stanza.MessageTypeChat
			}
			m := stanza.Message{
				Attrs: stanza.Attrs{
					To:   ActiveChats[0].Chat.jid,
					Type: typ, // FIXME: Change to MessageTypeGroupchat if isMuc is set to true
				},
				Body: msgEntry.Text,
			}
			client.Send(m)
			//msgEntry.SetText("")
		}
	})

	// Connect button
	connectBtn := widget.NewButton("Connect", func() {
		go func() {
			config := xmpp.Config{
				TransportConfiguration: xmpp.TransportConfiguration{
					Address: ActiveChats[0].Account.server,
				},
				Jid:          Accounts[0].jid,
				Credential:   xmpp.Password(Accounts[0].password),
				StreamLogger: os.Stdout,
				Insecure:     true,
			}

			router := xmpp.NewRouter()
			router.HandleFunc("message", func(s xmpp.Sender, p stanza.Packet) {
				msg, ok := p.(stanza.Message)
				if !ok {
					return
				}
				if msg.Type == stanza.MessageTypeChat && msg.Body != "" {
					fyne.DoAndWait(func() {
						chatLog.SetText(chatLog.Text + fmt.Sprintf("[%s] %s\n", msg.From, msg.Body))
					})
				}
			})

			c, err := xmpp.NewClient(&config, router, func(err error) {
				log.Println("Error:", err)
			})
			if err != nil {
				chatLog.SetText(chatLog.Text + fmt.Sprintf("❌ Connection failed: %v\n", err))
				return
			}

			client = c

			// Join MUC and request MAM history
			go func() {
				if ActiveChats[0].Chat.isMuc {
					time.Sleep(2 * time.Second)
					joinPresence := stanza.Presence{
						Attrs: stanza.Attrs{
							From: ActiveChats[0].Account.jid,
							To:   fmt.Sprintf("%s/%s", ActiveChats[0].Chat.jid, ActiveChats[0].Chat.nick),
						},
						Extensions: []stanza.PresExtension{
							stanza.MucPresence{},
						},
					}
					client.Send(joinPresence)
					chatLog.SetText(chatLog.Text + fmt.Sprintf("✅ Joined %s\n", ActiveChats[0].Chat.jid))

					time.Sleep(1 * time.Second)
					//requestMAMHistory(client, ActiveChats[0].Chat.jid, chatLog)
				}
			}()

			cm := xmpp.NewStreamManager(c, nil)
			cm.Run()
		}()
	})

	// Layout
	form := container.NewVBox(
		connectBtn,
		msgEntry,
		chatLog,
		sendBtn,
	)

	w.SetContent(form)
	w.ShowAndRun()
}

// requestMAMHistory sends a simple MAM query for the given room.
// FIXME does not work right now, lol
func requestMAMHistory(s xmpp.Sender, roomJID string, chatLog *widget.Entry) {
	// Basic MAM query IQ (latest messages)
	raw := `
	<iq type='set' id='mam1'>
			<query xmlns='urn:xmpp:mam:2' queryid='f27'/>
	</iq>`

	s.SendRaw(raw)
	chatLog.SetText(chatLog.Text + "(📜 Requested last 20 messages via MAM)\n")
}
