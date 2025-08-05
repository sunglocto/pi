<center>
<img src="https://github.com/sunglocto/pi/blob/255bc3749c089e3945871ddf19dd17d14a83f9ff/pi.png">
</center>

# π
[![Go](https://github.com/sunglocto/pi/actions/workflows/go.yml/badge.svg)](https://github.com/sunglocto/pi/actions/workflows/go.yml)
## the XMPP client from hell

Experimental and extremely weird XMPP client made with Go. No solicitors.

pi is currently pre-pre-pre-pre alpha software which you should not use right now.

pi uses [Fyne](https://fyne.io) for the frontend and uses the [Oasis SDK](https://github.com/jjj333-p/oasis-sdk) for XMPP functionality.

pi is an extremely opinionated client. It aims to have as little extra windows as possible, instead using alt-menus to perform many of the actions you'd see in a typical client.


## διαμόρφωση
(configuration)

When you launch pi, you will be greeted with a create account screen. You will then be able to enter your XMPP account details and then relaunch the application to log in.

If you want to add MUCs or DMs, you must configure the program by editing the pi.xml file. Here is the general idea:

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

Edit this file as necessary.

Currently joining and saving DM tabs is not supported, nor is getting avatars, reactions, encryption of media embed.

As of writing, pi supports basic message sending and receiving, replies and file upload.


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
vim pi.json
./pi
```
> Uh, Windows???

Eventually. Don't count on it.

Static executable snapshots are also provided for GNU/Linux systems.

## χρήση
(usage)

TODO

# επιπλέον
(extra)

Pi version numbers are the digits of Pi followed by a letter indicating the phase of development the program is in.

For example, the version string:

`3.14a`

Is the third version produced in the alpha phase.

The digits of Pi will reset back to `3` when moving to a new phase.

If the number gets too long, it will reset to one digit of 2π. Once that gets to long, it will be digits of 3π and etc.

Named after [Psi](https://github.com/psi-im/psi).
