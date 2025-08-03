package main

import (
	"fmt"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/dialog"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	oasisSdk "pain.agency/oasis-sdk"
)


func main() {

	login := oasisSdk.LoginInfo{
		Host: "sunglocto.net:5222",
		User: "bot2@sunglocto.net",
		Password: "iloverobots",
		DisplayName: "bot2",
		TLSoff: false,
		StartTLS: true,
		MucsToJoin: []string{"ringen@muc.isekai.rocks"},
	}

	maina := container.New(layout.NewHBoxLayout(), widget.NewLabel("pi"))
	scroller := container.NewVScroll(maina)

	client, err := oasisSdk.CreateClient(
		&login,
		func(client *oasisSdk.XmppClient, msg *oasisSdk.XMPPChatMessage) {
			fyne.Do(func(){
				card := widget.NewCard(msg.From.String(), *msg.CleanedBody, canvas.NewCircle(color.White))
				maina.Add(card)
				maina.Refresh()
				scroller.ScrollToBottom()
			})
		},
		func(client *oasisSdk.XmppClient, _ *muc.Channel, msg *oasisSdk.XMPPChatMessage) {
			fyne.Do(func(){
				if msg.Reply != nil {

				}
				card := widget.NewCard(msg.From.String(), *msg.CleanedBody, canvas.NewCircle(color.White))
				maina.Add(card)
				maina.Refresh()
				scroller.ScrollToBottom()
			})
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

	a := app.New()
	w := a.NewWindow("pi")
	w.Resize(fyne.NewSize(500,500))
	mit := fyne.NewMenuItem("About pi", func() {
		dialog.ShowInformation("About pi", "the XMPP client from hell\npi is an experimental XMPP client\nwritten by Sunglocto in Go.", w)
	})
	mib := fyne.NewMenuItem("Join a room", func() {
			nick := widget.NewEntry()
			room := widget.NewEntry()
		items := []*widget.FormItem{
			widget.NewFormItem("Nick", nick),
			widget.NewFormItem("MUC address", room),
		}

		dialog.ShowForm("Join a MUC", "Join", "Cancel", items, func(b bool) {
			if b {
				fmt.Println("attempting to join MUC")
				fmt.Println(nick)
				fmt.Println(room)
				go func(){
				client.MucClient.Join(client.Ctx, jid.MustParse(room.Text), client.Session, nil)
				}()
			}
		}, w)
	})
	menu_help := fyne.NewMenu("π", mit)
	menu_changeroom := fyne.NewMenu("β", mib)
	ma := fyne.NewMainMenu(menu_help, menu_changeroom)
	w.SetMainMenu(ma)

	tabs := container.NewAppTabs(container.NewTabItem("pi", widget.NewLabel("pi\nthe XMPP client from hell")), container.NewTabItem("chat", scroller))
	w.SetContent(tabs)
	w.ShowAndRun()
}
