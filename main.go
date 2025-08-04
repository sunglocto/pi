package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "fyne.io/x/fyne/theme"
	catppuccin "github.com/mbaklor/fyne-catppuccin"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/stanza"
	oasisSdk "pain.agency/oasis-sdk"
)

var version string = "3a"

// by sunglocto
// license AGPL

type Message struct {
	Author  string
	Content string
	ID      string
	ReplyID string
	ImageURL string
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
var connection bool = true

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
			icon := theme.FileImageIcon()
			btn := widget.NewButtonWithIcon("View image", icon, func() {

			})
			return container.NewVBox(author, content, btn)
		},
		func(i widget.ListItemID, co fyne.CanvasObject) {
			vbox := co.(*fyne.Container)
			author := vbox.Objects[0].(*widget.Label)
			content := vbox.Objects[1].(*widget.RichText)
			btn := vbox.Objects[2].(*widget.Button)
			btn.Hidden = true // Hide by default
			msgContent := tabData.Messages[i].Content
			if tabData.Messages[i].ImageURL != "" {
			btn.Hidden = false
			btn.OnTapped = func(){fyne.Do(func() {
					u, _ := storage.ParseURI(tabData.Messages[i].ImageURL)
					image := canvas.NewImageFromURI(u)
					image.FillMode = canvas.ImageFillOriginal
					dialog.ShowCustom("Image", "Close", image, w)
			})}
			}
			// Check if the message is a quote
			lines := strings.Split(msgContent, "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, ">") {
					lines[i] = "\n" + line + "\n"
				}
			}
			msgContent = strings.Join(lines, "\n")

			content.ParseMarkdown(msgContent)
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
		fyne.Do(func() {
		a = app.New()
		w = a.NewWindow("Error")
		w.Resize(fyne.NewSize(500, 500))
		dialog.ShowError(err, w)
		w.ShowAndRun()
		})
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
				var img string = ""
				if strings.Contains(str, "https://") {
					lines := strings.Split(str, " ")
					for i, line := range lines {
						s := strings.Split(line, " ")
						for j, v := range s {
							_, err := url.Parse(v)
							if err == nil && strings.HasPrefix(v, "https://") {
								img = v
								s[j] = fmt.Sprintf("[%s](%s)", v, v)
							}
						}
						lines[i] = strings.Join(s, " ")
					}
					str = strings.Join(lines, " ")
				}
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
					ImageURL: img,
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
					if strings.Contains(str, login.DisplayName) || (msg.Reply != nil && strings.Contains(msg.Reply.To, login.DisplayName)) {
						a.SendNotification(fyne.NewNotification(fmt.Sprintf("Mentioned in %s", mucJidStr), str))
					}
				}
				if strings.Contains(str, "https://") {
					lines := strings.Split(str, " ")
					for i, line := range lines {
						s := strings.Split(line, " ")
						for j, v := range s {
							_, err := url.Parse(v)
							if err == nil && strings.HasPrefix(v, "https://") {
								s[j] = fmt.Sprintf("[%s](%s)", v, v)
							}
						}
						lines[i] = strings.Join(s, " ")
					}
					str = strings.Join(lines, " ")
					fmt.Println(str)
				}
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
		for connection {
			err = client.Connect()
			if err != nil {
				responseChan := make(chan bool)
				fyne.Do(func() {
					dialog.ShowConfirm("disconnected", fmt.Sprintf("the client disconnected. would you like to try and reconnect?\nreason:\n%s", err.Error()), func(b bool) {
						responseChan <- b
					}, w)
				})
				if !<-responseChan {
					connection = false
				}
			}
		}
	}()

	a = app.New()
	a.Settings().SetTheme(myTheme{})
	w = a.NewWindow("pi")
	w.Resize(fyne.NewSize(500, 500))

	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Say something, you know you want to.")
	entry.OnChanged = func(s string) {
	}

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


	mit := fyne.NewMenuItem("about pi", func() {
		dialog.ShowInformation("about pi", fmt.Sprintf("the XMPP client from hell\n\npi is an experimental XMPP client\nwritten by Sunglocto in Go.\n\nVersion %s", version), w)
	})

	mia := fyne.NewMenuItem("configure message view", func() {
		ch := widget.NewCheck("", func(b bool) {})
		ch2 := widget.NewCheck("", func(b bool) {})
		ch.Checked = scrollDownOnNewMessage
		ch2.Checked = notifications
		scrollView := widget.NewFormItem("scroll to bottom on new message", ch)
		notiView := widget.NewFormItem("send notifications when mentioned", ch2)
		items := []*widget.FormItem{
			scrollView,
			notiView,
		}
		dialog.ShowForm("configure message view", "apply", "cancel", items, func(b bool) {
			if b {
				scrollDownOnNewMessage = ch.Checked
				notifications = ch2.Checked
			}
		}, w)
	})

	mis := fyne.NewMenuItem("clear chat window", func() {
		dialog.ShowConfirm("clear chat window", "are you sure you want to clear the chat window?", func(b bool) {
			if b {
				fmt.Println("clearing chat")
			}
		}, w)
	})
	/*mib := fyne.NewMenuItem("Join a room", func() {
		nickEntry := widget.NewEntry()
		nickEntry.SetText(login.DisplayName)
		roomEntry := widget.NewEntry()
		items := []*widget.FormItem{
			widget.NewFormItem("Nick", nickEntry),
			widget.NewFormItem("MUC address", roomEntry),
		}

		dialog.ShowForm("join a MUC", "join", "cancel", items, func(b bool) {
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
	})*/

	mic := fyne.NewMenuItem("upload a file", func() {
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
			dialog.ShowInformation("file successfully uploaded\nURL copied to your clipboard", link, w)
		}, w)
	})

	menu_help := fyne.NewMenu("π", mit)
	menu_changeroom := fyne.NewMenu("β", mic)
	menu_configureview := fyne.NewMenu("γ", mia, mis)
	bit := fyne.NewMenuItem("mark selected message as read", func() {
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

	bia := fyne.NewMenuItem("toggle replying to message", func() {
		replying = !replying
	})
	menu_messageoptions := fyne.NewMenu("Σ", bit, bia)
	ma := fyne.NewMainMenu(menu_help, menu_changeroom, menu_configureview, menu_messageoptions)
	w.SetMainMenu(ma)

	tabs = container.NewAppTabs(
		container.NewTabItem("τίποτα", widget.NewLabel(`
		welcome to pi

		you are currently not focused on any rooms.
		you can add new rooms by editing your pi.json file.
		in order to change application settings, refer to the tab-menu with the Greek letters. 
		these buttons allow you to configure the application as well as other functions.
		for more information about the pi project itself, hit the π button.
		`)),
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

	w.SetContent(container.NewVSplit(container.NewVSplit(tabs, container.NewHSplit(entry, sendbtn)), widget.NewLabel("pi")))
	w.ShowAndRun()
}
