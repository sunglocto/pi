# π
[![Go](https://github.com/sunglocto/pi/actions/workflows/go.yml/badge.svg)](https://github.com/sunglocto/pi/actions/workflows/go.yml)


Experimental and extremely weird XMPP client made with Go. No solicitors.

pi is currently pre-pre-pre-pre alpha software which you should not use right now.

pi uses [Fyne](https://fyne.io) for the frontend and uses the [Oasis SDK](https://github.com/jjj333-p/oasis-sdk) for XMPP functionality.

pi is an extremely opinionated client. It aims to have as little extra windows as possible, instead using alt-menus to perform many of the actions you'd see in a typical client.


## διαμόρφωση
(configuration)

In order to use pi, you currently have to create a `pi.json` file in the working directory of the executable. Here is how one looks like:

```json
{
    "Host":"example.com:5222",
    "User":"user@example.com",
    "Password":"123456",
    "DisplayName":"user",
    "NoTLS":false,
    "StartTLS":true,
    "Mucs":["room@muc.example.com"]
}
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
