<center>
<img src="https://github.com/sunglocto/pi/blob/255bc3749c089e3945871ddf19dd17d14a83f9ff/pi.png">
</center>

# π
[![build this now](https://github.com/sunglocto/pi/actions/workflows/go.yml/badge.svg)](https://github.com/sunglocto/pi/actions/workflows/go.yml)
<img width="1050" height="632" alt="image" src="https://github.com/user-attachments/assets/a5a3a7ab-8a85-49f0-81e9-ae5b977e7455" />

## the XMPP client from hell
> it's 10% code. 20% ai

Experimental and extremely weird XMPP client written in Go. No solicitors.

pi is currently pre-pre-pre-pre alpha software which you should not use as your primary XMPP client.

pi uses [Fyne](https://fyne.io) for the frontend and uses the [Oasis SDK](https://github.com/jjj333-p/oasis-sdk) for XMPP functionality.

pi is an extremely opinionated client. It aims to have as little extra windows as possible, instead using alt-menus to perform many of the actions you'd see in a typical client.


## διαμόρφωση
(configuration)

When you launch pi, you will be greeted with a create account screen. You will then be able to enter your XMPP account details and then relaunch the application to log in.

If you want to add MUCs or DMs, you must configure the program by editing the pi.xml file. Here is an example configuration:

```xml
<piConfig>
	<Login>
		<Host>example.com:5222</Host>
		<User>user@example.com</User>
		<Password>123456789</Password>
		<DisplayName>sunglocto</DisplayName>
		<TLSoff>false</TLSoff>
		<StartTLS>true</StartTLS>
		<MucsToJoin>room1@muc.example.com</MucsToJoin>
		<MucsToJoin>room2@muc.example.com</MucsToJoin>
	</Login>
	<Notifications>true</Notifications>
</piConfig>
```

Currently joining and saving DM tabs is not supported, nor is getting avatars, reactions or encryption.

As of writing, pi supports basic message sending and receiving, replies and ~~file upload~~.


## να χτίσω 
(building)

To build pi, you will need the latest version of Go, at least 1.21. You can grab it [here](https://go.dev).

The build instructions are very simple. Simply clone the repo, fetch the repositories and build the program:

Here is a summary of the commands you would need to use to build and run the program:
```bash
git clone https://github.com/sunglocto/pi
cd pi
go mod tidy
go build .
./pi
```
> Uh, Windows???

Eventually. Don't count on it.
Fyne has first-class support for Windows and none of my dependencies are platform dependent. I've built this app for Android before. If you compile it, it will most likely work with no issues.

Static executable snapshots are also provided for GNU/Linux systems, and CI runs on every commit, producing a binary on every successful build. You're welcome.


## υποστήριξη
(support)

You can file an issue and explain the problem you are having.

If you would like a more instant method of communication, join the [pi XMPP room.](xmpp:pi@room.sunglocto.net?join)

## μαρτυρίες
(testimonials)
From fellow insane and schizophrenic XMPP users:

> anyways this is your "just IM" client things ig.

> this looks like shit

> fyne is the best UI toolkit (sarcastic)

> i am going to explode you

> pi devstream when

## επιπλέον
(extra)

Pi version numbers are the digits of Pi followed by a letter indicating the phase of development the program is in.

For example, the version string:

`3.14a`

Is the third version produced in the alpha phase.

The digits of Pi will reset back to `3` when moving to a new phase.

If the number gets too long, it will reset to one digit of 2π. Once that gets too long, it will be digits of 3π and etc.

Named after [Psi](https://github.com/psi-im/psi).
