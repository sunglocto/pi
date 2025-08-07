package main

import (
	//core - required
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	// gui - required
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	// xmpp - required
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	oasisSdk "pain.agency/oasis-sdk"

	// gui - optional
	// catppuccin "github.com/mbaklor/fyne-catppuccin"
	adwaita "fyne.io/x/fyne/theme"
	// TODO: integrated theme switcher
)

var version string = "3.14a"
var statBar widget.Label
var chatInfo fyne.Container
var chatSidebar fyne.Container

// by sunglocto
// license AGPL

type Message struct {
	Author    string
	Content   string
	ID        string
	ReplyID   string
	ImageURL  string
	Raw       oasisSdk.XMPPChatMessage
	Important bool
}

type ChatTab struct {
	Jid                 jid.JID
	Nick                string
	Messages            []Message
	isMuc               bool
	Muc                 muc.Channel
	UpdateSidebar       bool
}

type ChatTabUI struct {
	Internal *ChatTab
	Scroller            *widget.List `xml:"-"`
	Sidebar             *fyne.Container `xml:"-"`
}

type piConfig struct {
	Login         oasisSdk.LoginInfo
	DMs           []string
	Notifications bool
}

var config piConfig
var login oasisSdk.LoginInfo
var DMs []string

var chatTabs = make(map[string]*ChatTab)
var UITabs = make(map[string]*ChatTabUI)

var AppTabs *container.AppTabs
var selectedId widget.ListItemID
var replying bool = false
var notifications bool
var connection bool = true

type myTheme struct{}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return adwaita.AdwaitaTheme().Color(name, variant)
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

func CreateUITab(chatJidStr string) ChatTabUI {
	var scroller *widget.List
	scroller = widget.NewList(
		func() int {
			return len(chatTabs[chatJidStr].Messages)
		},
		func() fyne.CanvasObject {
			author := widget.NewLabel("author")
			author.TextStyle.Bold = true
			content := widget.NewLabel("content")
			content.Wrapping = fyne.TextWrapWord
			content.Selectable = true
			icon := theme.FileVideoIcon()
			btn := widget.NewButtonWithIcon("View media", icon, func() {

			})
			return container.NewVBox(author, content, btn)
		},
		func(i widget.ListItemID, co fyne.CanvasObject) {
			vbox := co.(*fyne.Container)
			author := vbox.Objects[0].(*widget.Label)
			content := vbox.Objects[1].(*widget.Label)
			btn := vbox.Objects[2].(*widget.Button)
			if chatTabs[chatJidStr].Messages[i].Important {
				//content.Importance = widget.DangerImportance TODO: Fix highlighting messages with mentions, it's currently broken
			}
			btn.Hidden = true // Hide by default
			msgContent := chatTabs[chatJidStr].Messages[i].Content
			if chatTabs[chatJidStr].Messages[i].ImageURL != "" {
				btn.Hidden = false
				btn.OnTapped = func() {
					fyne.Do(func() {
						u, err := storage.ParseURI(chatTabs[chatJidStr].Messages[i].ImageURL)
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						if strings.HasSuffix(chatTabs[chatJidStr].Messages[i].ImageURL, "mp4") {
							url, err := url.Parse(chatTabs[chatJidStr].Messages[i].ImageURL)
							if err != nil {
								dialog.ShowError(err, w)
								return
							}
							a.OpenURL(url)
							return
						}
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

			//content.ParseMarkdown(msgContent)
			content.SetText(msgContent)
			if chatTabs[chatJidStr].Messages[i].ReplyID != "PICLIENT:UNAVAILABLE" {
				author.SetText(fmt.Sprintf("%s > %s", chatTabs[chatJidStr].Messages[i].Author, jid.MustParse(chatTabs[chatJidStr].Messages[i].Raw.Reply.To).Resourcepart()))
			} else {
				author.SetText(chatTabs[chatJidStr].Messages[i].Author)
			}
			scroller.SetItemHeight(i, vbox.MinSize().Height)
		},
	)



	scroller.OnSelected = func(id widget.ListItemID) {
		selectedId = id
	}

	myUITab := ChatTabUI{}

	scroller.CreateItem()
	myUITab.Scroller = scroller
	myUITab.Sidebar = container.NewVBox(widget.NewLabel("Data goes here")) 

	return myUITab
}
func addChatTab(isMuc bool, chatJid jid.JID, nick string) { 

	chatJidStr := chatJid.String()
	if _, ok := chatTabs[chatJidStr]; ok {
		// Tab already exists
		return
	}

	myChatTab := ChatTab{
		Jid:           chatJid,
		Nick:          nick,
		Messages:      []Message{},
		isMuc:         isMuc,
	}

	myUITab := CreateUITab(chatJid.String())
	myUITab.Internal = &myChatTab

	chatTabs[chatJidStr] = &myChatTab
	UITabs[chatJidStr] = &myUITab 

	fyne.Do(func() {
		AppTabs.Append(container.NewTabItem(chatJid.String(), myUITab.Scroller))
	})
}

func dropToSignInPage(reason string) {
	w = a.NewWindow("Welcome to Pi")
	w.Resize(fyne.NewSize(500, 500))
	rt := widget.NewRichTextFromMarkdown("# Welcome to pi\nIt appears you do not have a valid account configured. Let's create one!")
	footer := widget.NewRichTextFromMarkdown(fmt.Sprintf("Reason for being dropped to the sign-in page:\n\n```%s```", reason))
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
				config.Notifications = false

				bytes, err := xml.MarshalIndent(config, "", "\t")
				if err != nil {
					dialog.ShowError(err, w)
					return
				}

				writer, err := a.Storage().Create("pi.xml")
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				defer writer.Close()
				_, err = writer.Write(bytes)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				a.SendNotification(fyne.NewNotification("Done", "Relaunch the application"))
				a.Quit()
				//w.Close()
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
	muc.Since(time.Now())
	config = piConfig{}
	a = app.NewWithID("pi-im")
	reader, err := a.Storage().Open("pi.xml")
	if err != nil {
		dropToSignInPage(err.Error())
		return
	}
	defer reader.Close()

	bytes, err := io.ReadAll(reader)
	if err != nil {
		dropToSignInPage(err.Error())
		return
	}

	err = xml.Unmarshal(bytes, &config)
	if err != nil {
		dropToSignInPage(fmt.Sprintf("Your pi.xml file is invalid:\n%s", err.Error()))
		return
	}

	DMs = config.DMs
	login = config.Login
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
						for _, v := range s {
							_, err := url.Parse(v)
							if err == nil && strings.HasPrefix(v, "https://") {
								//s[j] = fmt.Sprintf("[%s](%s)", v, v)
								if strings.HasSuffix(v, ".png") || strings.HasSuffix(v, ".jpg") || strings.HasSuffix(v, ".jpeg") || strings.HasSuffix(v, ".webp") || strings.HasSuffix(v, ".mp4") {
									img = v
								}
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
					UITabs[userJidStr].Scroller.Refresh()
					if scrollDownOnNewMessage {
						UITabs[userJidStr].Scroller.ScrollToBottom()
					}
				})
			}
		},
		func(client *oasisSdk.XmppClient, muc *muc.Channel, msg *oasisSdk.XMPPChatMessage) {
			// HACK: IGNORING ALL MESSAGES FROM CLASSIC MUC HISTORY IN PREPARATION OF MAM SUPPORT
			ignore := false
			correction := false
			important := false
			for _, v := range msg.Unknown {
				if v.XMLName.Local == "delay" { // CLasic history message
					//ignore = true
					//fmt.Println("ignoring!")
				}
			}

			for _, v := range msg.Unknown {
				if v.XMLName.Local == "replace" {
					correction = true
				}
			}

			var ImageID string = ""
			mucJidStr := msg.From.Bare().String()
			if tab, ok := chatTabs[mucJidStr]; ok {
				chatTabs[mucJidStr].Muc = *muc
				str := *msg.CleanedBody
				if strings.Contains(str, login.DisplayName) {
					fmt.Println(str)
					important = true
				}
				if !ignore && notifications {
					if !correction && msg.From.String() != client.JID.String() && strings.Contains(str, login.DisplayName) || (msg.Reply != nil && strings.Contains(msg.Reply.To, login.DisplayName)) {
						a.SendNotification(fyne.NewNotification(fmt.Sprintf("Mentioned in %s", mucJidStr), str))
					}
				}
				if strings.Contains(str, "https://") {
					lines := strings.Split(str, "\n")
					for i, line := range lines {
						s := strings.Split(line, " ")
						for _, v := range s {
							_, err := url.Parse(v)
							if err == nil && strings.HasPrefix(v, "https://") {
								//s[j] = fmt.Sprintf("[%s](%s)", v, v)
								if strings.HasSuffix(v, ".png") || strings.HasSuffix(v, ".jpg") || strings.HasSuffix(v, ".jpeg") || strings.HasSuffix(v, ".webp") || strings.HasSuffix(v, ".mp4") {
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

				if correction {
					for i := len(tab.Messages) - 1; i > 0; i-- {
						if tab.Messages[i].Raw.From.String() == msg.From.String() {
							tab.Messages[i].Content = *msg.CleanedBody + " (edited)"
							fyne.Do(func() {
								UITabs[mucJidStr].Scroller.Refresh()
							})
							return
						}
					}
				}

				myMessage := Message{
					Author:    msg.From.Resourcepart(),
					Content:   str,
					ID:        msg.ID,
					ReplyID:   replyID,
					Raw:       *msg,
					ImageURL:  ImageID,
					Important: important,
				}
				if !ignore {
					tab.Messages = append(tab.Messages, myMessage)
				}
				fyne.Do(func() {
					UITabs[mucJidStr].Scroller.Refresh()
					if scrollDownOnNewMessage {
						UITabs[mucJidStr].Scroller.ScrollToBottom()
					}
				})
			}
		},
		func(_ *oasisSdk.XmppClient, from jid.JID, state oasisSdk.ChatState) {
			switch state {
			case oasisSdk.ChatStateComposing:
				fyne.Do(func() {
					statBar.SetText(fmt.Sprintf("%s is typing...", from.Resourcepart()))
				})
			case oasisSdk.ChatStatePaused:

				fyne.Do(func() {
					statBar.SetText(fmt.Sprintf("%s has stoped typing.", from.Resourcepart()))
				})
			case oasisSdk.ChatStateInactive:
				fyne.Do(func() {
					statBar.SetText(fmt.Sprintf("%s is idle", from.Resourcepart()))
				})
			case oasisSdk.ChatStateGone:
				fyne.Do(func() {
					statBar.SetText(fmt.Sprintf("%s is gone", from.Resourcepart()))
				})
			default:
				fyne.Do(func() {
					statBar.SetText("")
				})
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

	SendCallback := func() {
		text := entry.Text
		if AppTabs.Selected() == nil || AppTabs.Selected().Content == nil || text == "" {
			return
		}

		selectedScroller, ok := AppTabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}

		var activeMucJid string
		var isMuc bool
		for jid, tabData := range UITabs {
			if tabData.Scroller == selectedScroller {
				activeMucJid = jid
				isMuc = chatTabs[activeMucJid].isMuc
				break
			}
		}

		if activeMucJid == "" {
			return
		}

		go func() {
			if replying {
				m := chatTabs[activeMucJid].Messages[selectedId].Raw
				err = client.ReplyToEvent(&m, text)
				if err != nil {
					dialog.ShowError(err, w)
				}
				return
			}

			url, uerr := url.Parse(strings.Split(text, " ")[0])
			if uerr == nil && strings.HasPrefix(strings.Split(text, " ")[0], "https://") {
				err = client.SendImage(jid.MustParse(activeMucJid).Bare(), text, url.String(), &text)
				if err != nil {
					dialog.ShowError(err, w)
				}
				return
			}
			err = client.SendText(jid.MustParse(activeMucJid).Bare(), text)

			if err != nil {
				dialog.ShowError(err, w)
			}
		}()

		if !isMuc {
			chatTabs[activeMucJid].Messages = append(chatTabs[activeMucJid].Messages, Message{
				Author:  "You",
				Content: text,
				ReplyID: "PICLIENT:UNAVAILABLE",
			})
			fyne.Do(func() {
				if scrollDownOnNewMessage {
					UITabs[activeMucJid].Scroller.ScrollToBottom()
				}
			})
		}
		entry.SetText("")
	}

	sendbtn := widget.NewButton("Send", SendCallback)
	entry.OnSubmitted = func(s string) {
		SendCallback()
		// i fucking hate fyne
	}

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
		selectedScroller, ok := AppTabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}
		selectedScroller.ScrollToBottom()
	})

	jtt := fyne.NewMenuItem("jump to top", func() {
		selectedScroller, ok := AppTabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}
		selectedScroller.ScrollToTop()
	})

	w.SetOnDropped(func(p fyne.Position, u []fyne.URI) {
		var link string
		myUri := u[0] // Only upload a single file
		progress := make(chan oasisSdk.UploadProgress)
		myprogressbar := widget.NewProgressBar()
		diag := dialog.NewCustom("Uploading file", "Hide", myprogressbar, w)
		diag.Show()
		go func() {
			client.UploadFile(client.Ctx, myUri.Path(), progress)
		}()

		for update := range progress {
			fyne.Do(func() {
				myprogressbar.Value = float64(update.Percentage) / 100
				myprogressbar.Refresh()
			})

			if update.Error != nil {
				diag.Dismiss()
				dialog.ShowError(update.Error, w)
				return
			}

			if update.GetURL != "" {
				link = update.GetURL
			}
		}

		diag.Dismiss()
		a.Clipboard().SetContent(link)
		dialog.ShowInformation("file successfully uploaded\nURL copied to your clipboard", link, w)

	})

	//deb := fyne.NewMenuItem("DEBUG: Attempt to get MAM history from a user", func() {
		//res, err := history.Fetch(client.Ctx, history.Query{}, jid.MustParse("ringen@muc.isekai.rocks"), client.Session)
	//})
	mic := fyne.NewMenuItem("upload a file", func() {
		var link string
		var toperr error
		//var topreader fyne.URIReadCloser
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			go func() {

				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				if reader == nil {
					return
				}
				bytes, toperr = io.ReadAll(reader)
				//topreader = reader

				if toperr != nil {
					dialog.ShowError(toperr, w)
					return
				}

				progress := make(chan oasisSdk.UploadProgress)
				myprogressbar := widget.NewProgressBar()
				diag := dialog.NewCustom("Uploading file", "Hide", myprogressbar, w)
				diag.Show()
				go func() {
					client.UploadFile(client.Ctx, reader.URI().Path(), progress)
				}()

				for update := range progress {
					fyne.Do(func() {
						myprogressbar.Value = float64(update.Percentage) / 100
						myprogressbar.Refresh()
					})

					if update.Error != nil {
						diag.Dismiss()
						dialog.ShowError(update.Error, w)
						return
					}

					if update.GetURL != "" {
						link = update.GetURL
					}
				}

				diag.Dismiss()
				a.Clipboard().SetContent(link)
				dialog.ShowInformation("file successfully uploaded\nURL copied to your clipboard", link, w)
			}()

		}, w)
	})

	servDisc := fyne.NewMenuItem("Disco features", func() {
		var search jid.JID
		dialog.ShowEntryDialog("Disco features", "JID: ", func(s string) { // TODO: replace with undeprecated widget
			search, err = jid.Parse(s)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			myBox := container.NewGridWithColumns(1, widget.NewLabel("Items\na\na\na\na\na"))
			info, err := disco.GetInfo(client.Ctx, "", search, client.Session)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			m := info.Features
			for _, v := range m {
				myBox.Add(widget.NewLabel(v.Var))
				myBox.Refresh()
			}

			dialog.ShowCustom("Features", "cancel", myBox, w)

		}, w)
	})


	savedata := fyne.NewMenuItem("DEBUG: Save tab data to disk", func() {
		d := []ChatTab{}
		for _, v := range chatTabs {
			d = append(d, *v)
		}
		b, err := xml.Marshal(d)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		os.Create("test.xml")
		os.WriteFile("text.xml", b, os.ModeAppend)
	})
	menu_help := fyne.NewMenu("π", mit, reconnect, savedata)
	menu_changeroom := fyne.NewMenu("β", mic, servDisc)
	menu_configureview := fyne.NewMenu("γ", mia, mis, jtt, jtb)
	hafjag := fyne.NewMenuItem("Hafjag", func() {
		entry.Text = "Hafjag"
		SendCallback()
		entry.Text = "Hafjag"
	})

	hotfuck := fyne.NewMenuItem("Hot Fuck", func() {
		entry.Text = "Hot Fuck"
		SendCallback()
		entry.Text = "Oh Yeah."
	})

	mycurrenttime := fyne.NewMenuItem("Current time", func() {
		entry.Text = fmt.Sprintf("It is currently %s", time.Now().Format(time.RFC850))
		SendCallback()
	})
	menu_jokes := fyne.NewMenu("Δ", mycurrenttime, hafjag, hotfuck)
	bit := fyne.NewMenuItem("mark selected message as read", func() {
		selectedScroller, ok := AppTabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}
		var activeMucJid string
		for jid, tabData := range UITabs {
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

		selectedScroller, ok := AppTabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}

		var activeChatJid string
		for jid, tabData := range UITabs {
			if tabData.Scroller == selectedScroller {
				activeChatJid = jid
				break
			}
		}

		m := chatTabs[activeChatJid].Messages[selectedId].Raw
		bytes, err := xml.MarshalIndent(m, "", "\t")
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
	ma := fyne.NewMainMenu(menu_help, menu_changeroom, menu_configureview, menu_messageoptions, menu_jokes)
	w.SetMainMenu(ma)

	AppTabs = container.NewAppTabs(
		container.NewTabItem("τίποτα", widget.NewLabel(`
		pi
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

	AppTabs.OnSelected = func(ti *container.TabItem) {
		selectedScroller, ok := AppTabs.Selected().Content.(*widget.List)
		if !ok {
			return
		}

		var activeChatJid string
		for jid, tabData := range UITabs {
			if tabData.Scroller == selectedScroller {
				activeChatJid = jid
				break
			}
		}



		tab := chatTabs[activeChatJid]
		UITab := UITabs[activeChatJid]
		if tab.isMuc {
			chatInfo = *container.NewHBox(widget.NewLabel(tab.Muc.Addr().String()))
		} else {
			chatInfo = *container.NewHBox(widget.NewLabel(tab.Jid.String()))
		}

		chatSidebar = *UITab.Sidebar
		chatSidebar.Refresh()
	}

	statBar.SetText("")
	w.SetContent(container.NewVSplit(container.NewVSplit(container.NewHSplit(AppTabs, &chatSidebar), container.NewHSplit(entry, sendbtn)), container.NewHSplit(&statBar, &chatInfo)))
	w.ShowAndRun()
}
