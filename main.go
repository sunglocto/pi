package main

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	_ "io/fs"
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
	catppuccin "github.com/mbaklor/fyne-catppuccin"
	_ "fyne.io/x/fyne/theme"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	oasisSdk "pain.agency/oasis-sdk"
)

var version string = "3.1a"

// by sunglocto
// license AGPL

type Message struct {
	Author   string
	Content  string
	ID       string
	ReplyID  string
	ImageURL string
	Raw      oasisSdk.XMPPChatMessage
}

type MucTab struct {
	Jid      jid.JID
	Nick     string
	Messages []Message
	Scroller *widget.List
	isMuc    bool
}

type piConfig struct {
	Login         oasisSdk.LoginInfo
	DMs           []string
	Notifications bool
}

var config piConfig
var login oasisSdk.LoginInfo
var DMs []string

var chatTabs = make(map[string]*MucTab)
var tabs *container.AppTabs
var selectedId widget.ListItemID
var replying bool = false
var notifications bool
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
				btn.OnTapped = func() {
					fyne.Do(func() {
						u, _ := storage.ParseURI(tabData.Messages[i].ImageURL)
						image := canvas.NewImageFromURI(u)
						image.FillMode = canvas.ImageFillOriginal
						dialog.ShowCustom("Image", "Close", image, w)
					})
				}
			}
			// Check if the message is a quote
			lines := strings.Split(msgContent, "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, ">") {
					lines[i] = fmt.Sprintf("\n %s \n", line)
				}
			}
			msgContent = strings.Join(lines, "\n")

			content.ParseMarkdown(msgContent)
			if tabData.Messages[i].ReplyID != "PICLIENT:UNAVAILABLE" {
				author.SetText(fmt.Sprintf("%s > %s", tabData.Messages[i].Author, jid.MustParse(tabData.Messages[i].ReplyID).Resourcepart()))
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

func dropToSignInPage(reason string) {
	a = app.New()
	w = a.NewWindow("Welcome to Pi")
	w.Resize(fyne.NewSize(500, 500))
	rt := widget.NewRichTextFromMarkdown("# Welcome to pi\nIt appears you do not have a valid account configured. Let's create one!")
	footer := widget.NewRichTextFromMarkdown(fmt.Sprintf("Reason for being dropped to the sign-in page:\n\n```%s```\n\nDEBUG: %s", reason, fmt.Sprint(os.DirFS("."))))
	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("Your JID")
	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("Server and port")
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Your Password")
	nicknameEntry := widget.NewEntry()
	nicknameEntry.SetPlaceHolder("Your Nickname")

	userView := widget.NewFormItem("", userEntry)
	serverView := widget.NewFormItem("", serverEntry)
	passwordView := widget.NewFormItem("", passwordEntry)
	nicknameView := widget.NewFormItem("", nicknameEntry)
	items := []*widget.FormItem{
		serverView,
		userView,
		passwordView,
		nicknameView,
	}

	btn := widget.NewButton("Create an account", func() {
		dialog.ShowForm("Create an account", "Create", "Dismiss", items, func(b bool) {
			if b {
				config := piConfig{}
				config.Login.Host = serverEntry.Text
				config.Login.User = userEntry.Text
				config.Login.Password = passwordEntry.Text
				config.Login.DisplayName = nicknameEntry.Text
				config.Notifications = true

				bytes, err := xml.MarshalIndent(config, "", "	")
				if err != nil {
					dialog.ShowError(err, w)
					return
				}

				_, err = os.Create("pi.xml")
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				err = os.WriteFile("pi.xml", bytes, os.FileMode(os.O_RDWR)) // TODO: See if this works on non-unix like systems
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				a.SendNotification(fyne.NewNotification("Done", "Relaunch the application"))
				w.Close()
			}
		}, w)
	})
	btn2 := widget.NewButton("Close pi", func() {
		w.Close()
	})
	w.SetContent(container.NewVBox(rt, btn, btn2, footer))
	w.ShowAndRun()

}

func main() {

	config = piConfig{}

	bytes, err := os.ReadFile("./pi.xml")
	if err != nil {
		dropToSignInPage(err.Error())
		return
	}

	err = xml.Unmarshal(bytes, &config)
	if err != nil {
		dropToSignInPage(fmt.Sprintf("Your pi.xml file is invalid:\n%s", err.Error()))
		return
	}

	login = config.Login
	DMs = config.DMs
	notifications = config.Notifications

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
					lines := strings.Split(str, "\n")
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
					Author:   msg.From.Resourcepart(),
					Content:  str,
					ID:       msg.ID,
					ReplyID:  replyID,
					Raw:      *msg,
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
			var ImageID string = ""
			mucJidStr := msg.From.Bare().String()
			if tab, ok := chatTabs[mucJidStr]; ok {

				str := *msg.CleanedBody
				if notifications {
					if strings.Contains(str, login.DisplayName) || (msg.Reply != nil && strings.Contains(msg.Reply.To, login.DisplayName)) {
						a.SendNotification(fyne.NewNotification(fmt.Sprintf("Mentioned in %s", mucJidStr), str))
					}
				}
				if strings.Contains(str, "https://") {
					lines := strings.Split(str, "\n")
					for i, line := range lines {
						s := strings.Split(line, " ")
						for j, v := range s {
							_, err := url.Parse(v)
							if err == nil && strings.HasPrefix(v, "https://") {
								s[j] = fmt.Sprintf("[%s](%s)", v, v)
								if strings.HasSuffix(v, ".png") || strings.HasSuffix(v, ".jp") || strings.HasSuffix(v, ".webp") {
									ImageID = v
								}
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
					Author:   msg.From.Resourcepart(),
					Content:  str,
					ID:       msg.ID,
					ReplyID:  replyID,
					Raw:      *msg,
					ImageURL: ImageID,
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

			err = client.SendText(jid.MustParse(activeMucJid), text)
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

	reconnect := fyne.NewMenuItem("reconnect", func() {
		go func() {
			err := client.Connect()
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(err, w)
				})
			}
		}()
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

	jtb := fyne.NewMenuItem("jump to bottom", func() {
		selectedScroller, ok := tabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}
		selectedScroller.ScrollToBottom()
	})

	jtt := fyne.NewMenuItem("jump to top", func() {
		selectedScroller, ok := tabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}
		selectedScroller.ScrollToTop()
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
		var link string
		var bytes []byte
		var toperr error
		var topreader fyne.URIReadCloser
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}
			bytes, toperr = io.ReadAll(reader)
			topreader = reader

			if toperr != nil {
				dialog.ShowError(toperr, w)
				return
			}

			progress := make(chan oasisSdk.UploadProgress)
			myprogressbar := widget.NewProgressBar()
			dialog.ShowCustom("Uploading file", "Hide", myprogressbar, w)
			go func() {

				client.UploadFileFromBytes(client.Ctx, topreader.URI().Name(), bytes, progress)
			}()
			for update := range progress {
				myprogressbar.Value = float64(update.Percentage)
				myprogressbar.Refresh()

				if update.Error != nil {
					dialog.ShowError(update.Error, w)
					return
				}

				if update.GetURL != "" {
					link = update.GetURL
				}
			}

			a.Clipboard().SetContent(link)
			dialog.ShowInformation("file successfully uploaded\nURL copied to your clipboard", link, w)

		}, w)
	})

	menu_help := fyne.NewMenu("π", mit, reconnect)
	menu_changeroom := fyne.NewMenu("β", mic)
	menu_configureview := fyne.NewMenu("γ", mia, mis, jtt, jtb)
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

	bic := fyne.NewMenuItem("show message XML", func() {
		pre := widget.NewLabel("")

		selectedScroller, ok := tabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}

		var activeChatJid string
		for jid, tabData := range chatTabs {
			if tabData.Scroller == selectedScroller {
				activeChatJid = jid
				break
			}
		}

		m := chatTabs[activeChatJid].Messages[selectedId].Raw
		bytes, err := xml.MarshalIndent(m, "", "	")
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		pre.SetText(string(bytes))
		pre.Selectable = true
		pre.Refresh()
		dialog.ShowCustom("Message", "Close", pre, w)
	})
	menu_messageoptions := fyne.NewMenu("Σ", bit, bia, bic)
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
