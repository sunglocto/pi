package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	_ "net/url"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	_ "fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "fyne.io/x/fyne/theme"
	catppuccin "github.com/mbaklor/fyne-catppuccin"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/stanza"
	oasisSdk "pain.agency/oasis-sdk"
)

// by sunglocto
// license AGPL

type Message struct {
	Author  string
	Content string
	ID      string
	ReplyID string
	Raw     oasisSdk.XMPPChatMessage
}

type MucTab struct {
	Jid      jid.JID
	Nick     string
	Messages []Message
	Scroller *widget.List
	isMuc    bool
}

var chatTabs = make(map[string]*MucTab)
var tabs *container.AppTabs
var selectedId widget.ListItemID
var replying bool = false
var notifications bool = true

type myTheme struct{}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return catppuccin.New().Color(name, variant)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameHeadingText {
		return 18
	}
	return theme.DefaultTheme().Size(name)
}

var scrollDownOnNewMessage bool = true
var w fyne.Window
var a fyne.App

func addChatTab(isMuc bool, chatJid jid.JID, nick string) {
	mucJidStr := chatJid.String()
	if _, ok := chatTabs[mucJidStr]; ok {
		// Tab already exists
		return
	}

	tabData := &MucTab{
		Jid:      chatJid,
		Nick:     nick,
		Messages: []Message{},
		isMuc:    isMuc,
	}

	var scroller *widget.List
	scroller = widget.NewList(
		func() int {
			return len(tabData.Messages)
		},
		func() fyne.CanvasObject {
			author := widget.NewLabel("author")
			author.TextStyle.Bold = true
			content := widget.NewRichTextWithText("content")
			content.Wrapping = fyne.TextWrapWord
			return container.NewVBox(author, content)
		},
		func(i widget.ListItemID, co fyne.CanvasObject) {
			vbox := co.(*fyne.Container)
			author := vbox.Objects[0].(*widget.Label)
			content := vbox.Objects[1].(*widget.RichText)
			content.ParseMarkdown(tabData.Messages[i].Content)
			if tabData.Messages[i].ReplyID != "PICLIENT:UNAVAILABLE" {
				author.SetText(fmt.Sprintf("%s ↳ %s", tabData.Messages[i].Author, jid.MustParse(tabData.Messages[i].ReplyID).Resourcepart()))
			} else {
				author.SetText(tabData.Messages[i].Author)
			}
			scroller.SetItemHeight(i, vbox.MinSize().Height)
		},
	)
	scroller.OnSelected = func(id widget.ListItemID) {
		selectedId = id
	}
	tabData.Scroller = scroller

	chatTabs[mucJidStr] = tabData

	tabItem := container.NewTabItem(chatJid.Localpart(), scroller)
	tabs.Append(tabItem)
}

func main() {
	login := oasisSdk.LoginInfo{}

	DMs := []string{}

	bytes, err := os.ReadFile("./pi.json")
	if err != nil {
		a = app.New()
		w = a.NewWindow("Error")
		w.Resize(fyne.NewSize(500, 500))
		dialog.ShowInformation("Error", fmt.Sprintf("Please make sure there is a file named pi.json in the same directory you are running this executable...\n%s", err.Error()), w)
		w.ShowAndRun()
		return
	}
	err = json.Unmarshal(bytes, &login)
	if err != nil {
		a = app.New()
		w = a.NewWindow("Error")
		w.Resize(fyne.NewSize(500, 500))
		dialog.ShowError(err, w)
		w.ShowAndRun()
		return
	}

	client, err := oasisSdk.CreateClient(
		&login,
		func(client *oasisSdk.XmppClient, msg *oasisSdk.XMPPChatMessage) {
			fmt.Println(msg)
			userJidStr := msg.From.Bare().String()
			tab, ok := chatTabs[userJidStr]
			fmt.Println(msg.From.String())
			if ok {
				str := *msg.CleanedBody
				if notifications {
					a.SendNotification(fyne.NewNotification(fmt.Sprintf("%s says", userJidStr), str))
				}
				/*
					if strings.Contains(str, "https://") {
							fmt.Println("Attempting to do URL thingy")
							s := strings.Split(str, " ")
							for i, v := range s {
								_, err := url.Parse(v)
								if err == nil {
									s[i] = fmt.Sprintf("[%s](%s)", v, v)
								}
							}
							str = strings.Join(s, " ")
					}*/
				var replyID string
				if msg.Reply == nil {
					replyID = "PICLIENT:UNAVAILABLE"
				} else {
					replyID = msg.Reply.ID
				}
				myMessage := Message{
					Author:  msg.From.Resourcepart(),
					Content: str,
					ID:      msg.ID,
					ReplyID: replyID,
					Raw:     *msg,
				}

				tab.Messages = append(tab.Messages, myMessage)
				fyne.Do(func() {
					tab.Scroller.Refresh()
					if scrollDownOnNewMessage {
						tab.Scroller.ScrollToBottom()
					}
				})
			}
		},
		func(client *oasisSdk.XmppClient, _ *muc.Channel, msg *oasisSdk.XMPPChatMessage) {
			mucJidStr := msg.From.Bare().String()
			if tab, ok := chatTabs[mucJidStr]; ok {

				str := *msg.CleanedBody
				if notifications {
					if strings.Contains(str, login.DisplayName) || (msg.Reply != nil && strings.Contains(msg.Reply.To, login.User)) {
						a.SendNotification(fyne.NewNotification(fmt.Sprintf("Mentioned in %s", mucJidStr), str))
					}
				}
				/*
					if strings.Contains(str, "https://") {
							s := strings.Split(str, " ")
							for i, v := range s {
								_, err := url.Parse(v)
								if err == nil {
									s[i] = fmt.Sprintf("[%s](%s)", v, v)
								}
							}
							str = strings.Join(s, " ")
					}*/
				fmt.Println(msg.ID)
				var replyID string
				if msg.Reply == nil {
					replyID = "PICLIENT:UNAVAILABLE"
				} else {
					replyID = msg.Reply.To
				}
				myMessage := Message{
					Author:  msg.From.Resourcepart(),
					Content: str,
					ID:      msg.ID,
					ReplyID: replyID,
					Raw:     *msg,
				}
				tab.Messages = append(tab.Messages, myMessage)
				fyne.Do(func() {
					tab.Scroller.Refresh()
					if scrollDownOnNewMessage {
						tab.Scroller.ScrollToBottom()
					}
				})
			}
		},
		func(_ *oasisSdk.XmppClient, from jid.JID, state oasisSdk.ChatState) {
			//fromStr := from.String()
			switch state {
			case oasisSdk.ChatStateActive:
			case oasisSdk.ChatStateComposing:
			case oasisSdk.ChatStatePaused:
			case oasisSdk.ChatStateInactive:
			case oasisSdk.ChatStateGone:
			default:
			}
		},
		func(_ *oasisSdk.XmppClient, from jid.JID, id string) {
			fmt.Printf("Delivered %s to %s", id, from.String())
		},
		func(_ *oasisSdk.XmppClient, from jid.JID, id string) {
			fmt.Printf("%s has seen %s", from.String(), id)
		},
	)

	if err != nil {
		log.Fatalln("Could not create client - " + err.Error())
	}

	go func() {
		err = client.Connect()
		if err != nil {
			log.Fatalln("Could not connect - " + err.Error())
		}
	}()

	a = app.New()
	a.Settings().SetTheme(myTheme{})
	w = a.NewWindow("pi")
	w.Resize(fyne.NewSize(500, 500))

	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Say something, you know you want to.")

	sendbtn := widget.NewButton("Send", func() {
		text := entry.Text
		if tabs.Selected() == nil || tabs.Selected().Content == nil {
			return
		}

		selectedScroller, ok := tabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}

		var activeMucJid string
		var isMuc bool
		for jid, tabData := range chatTabs {
			if tabData.Scroller == selectedScroller {
				activeMucJid = jid
				isMuc = tabData.isMuc
				break
			}
		}

		if activeMucJid == "" {
			return
		}

		go func() {
			//TODO: Fix message hack until jjj adds message sending
			if replying {
				m := chatTabs[activeMucJid].Messages[selectedId].Raw
				client.ReplyToEvent(&m, text)
				return
			}
			var typ stanza.MessageType
			if isMuc {
				typ = stanza.GroupChatMessage
			} else {
				typ = stanza.ChatMessage
			}
			msg := oasisSdk.XMPPChatMessage{
				Message: stanza.Message{
					To:   jid.MustParse(activeMucJid),
					Type: typ,
				},
				ChatMessageBody: oasisSdk.ChatMessageBody{
					Body: &text,
				},
			}
			err := client.Session.Encode(client.Ctx, msg)
			if err != nil {
				dialog.ShowError(err, w)
			}
		}()

		if !isMuc {
			chatTabs[activeMucJid].Messages = append(chatTabs[activeMucJid].Messages, Message{
				Author:  "You",
				Content: text,
			})
			fyne.Do(func() {
				if scrollDownOnNewMessage {
					chatTabs[activeMucJid].Scroller.ScrollToBottom()
				}
			})
		}
		entry.SetText("")
	})

	mit := fyne.NewMenuItem("About pi", func() {
		dialog.ShowInformation("About pi", "the XMPP client from hell\n\npi is an experimental XMPP client\nwritten by Sunglocto in Go.", w)
	})

	mia := fyne.NewMenuItem("Configure message view", func() {
		ch := widget.NewCheck("", func(b bool) {})
		ch2 := widget.NewCheck("", func(b bool) {})
		ch.Checked = scrollDownOnNewMessage
		ch2.Checked = notifications
		scrollView := widget.NewFormItem("Scroll to bottom on new message", ch)
		notiView := widget.NewFormItem("Send notifications when mentioned", ch2)
		items := []*widget.FormItem{
			scrollView,
			notiView,
		}
		dialog.ShowForm("Configure message view", "Apply", "Cancel", items, func(b bool) {
			if b {
				scrollDownOnNewMessage = ch.Checked
				notifications = ch2.Checked
			}
		}, w)
	})

	mis := fyne.NewMenuItem("Clear chat window", func() {
		dialog.ShowConfirm("Clear chat window", "Are you sure you want to clear the chat window?", func(b bool) {
			if b {
				fmt.Println("clearing chat")
			}
		}, w)
	})
	mib := fyne.NewMenuItem("Join a room", func() {
		nickEntry := widget.NewEntry()
		nickEntry.SetText(login.DisplayName)
		roomEntry := widget.NewEntry()
		items := []*widget.FormItem{
			widget.NewFormItem("Nick", nickEntry),
			widget.NewFormItem("MUC address", roomEntry),
		}

		dialog.ShowForm("Join a MUC", "Join", "Cancel", items, func(b bool) {
			if b {
				roomJid, err := jid.Parse(roomEntry.Text)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				nick := nickEntry.Text
				go func() {
					// We probably don't need to handle the error here, if it fails the user will know
					_, err := client.MucClient.Join(client.Ctx, roomJid, client.Session, nil)
					if err != nil {
						panic(err)
					}
				}()
				addChatTab(true, roomJid, nick)
			}
		}, w)
	})

	mic := fyne.NewMenuItem("Upload a file", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
			}
			bytes, err := io.ReadAll(reader)
			link, err := client.UploadFileFromBytes(reader.URI().String(), bytes)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			a.Clipboard().SetContent(link)
			dialog.ShowInformation("File successfully uploaded", link, w)
		}, w)
	})

	menu_help := fyne.NewMenu("π", mit)
	menu_changeroom := fyne.NewMenu("β", mib, mic)
	menu_configureview := fyne.NewMenu("γ", mia, mis)
	bit := fyne.NewMenuItem("Mark message as read", func() {
		selectedScroller, ok := tabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}
		var activeMucJid string
		for jid, tabData := range chatTabs {
			if tabData.Scroller == selectedScroller {
				activeMucJid = jid
				break
			}
		}

		m := chatTabs[activeMucJid].Messages[selectedId].Raw
		client.MarkAsRead(&m)
	})

	bia := fyne.NewMenuItem("Toggle replying to message", func() {
		replying = !replying
	})
	menu_messageoptions := fyne.NewMenu("Σ", bit, bia)
	ma := fyne.NewMainMenu(menu_help, menu_changeroom, menu_configureview, menu_messageoptions)
	w.SetMainMenu(ma)

	tabs = container.NewAppTabs(
		container.NewTabItem("τίποτα", widget.NewRichTextFromMarkdown("# No chat selected.")),
	)

	for _, mucJidStr := range login.MucsToJoin {
		mucJid, err := jid.Parse(mucJidStr)
		if err == nil {
			addChatTab(true, mucJid, login.DisplayName)
		}
	}

	for _, userJidStr := range DMs {
		fmt.Println(userJidStr)
		DMjid, err := jid.Parse(userJidStr)
		if err == nil {
			addChatTab(false, DMjid, login.DisplayName)
		}
	}

	w.SetContent(container.NewVSplit(tabs, container.NewHSplit(entry, sendbtn)))
	w.ShowAndRun()
}
