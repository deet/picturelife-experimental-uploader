# Picturelife Experimental Uploader

A cross platform Picturelife media uploader.

## IMPORTANT NOTE

This project was an EXPERIMENTAL PROTOTYPE. IT IS NOT SUPPORTED OR MAINTAINED by Picturelife or Keith Brisson.

Picturelife intends to release a stable cross platform uploader as soon as possible. Stay tuned.

As a prototype, it *should* work but may not. It is provided for academic purposes and as a stopgap for highly technical user who need to upload to Picturelife from currently unsupported platforms. 

This project was a proof of concept. The source code is rough and there are no tests.

Known issues:
- Login is required from the command line
- It does not daemonize itself or start automatically
- The UI gets slow as your uploaded media number increases
- It does not gracefully retry failed uploads
- Logging is ugly and verbose
- Error handling results in ugly log messages

It has been tested on Ubuntu, Windows, and OS X. Success is not guaranteed. And it was developed and tested primarily on Ubuntu. Currently, the OS X build will only run if you disable the tray icon. To build on Windows, you need to create an icon file (see the documentation for the Trayhost import).

# Binary versions

See picturelife.com/download/experimental

## Requirements

- Go 1.1 (http://golang.org/doc/install)
- API credentials
- A browser with Websocket support to use the GUI

## How to setup

1. Clone the repository.
2. At the root of repository, run 'go get' to fetch dependencies. If this fails, make sure you have set your GOPATH.

## Get API credentials

Contact Picturelife to get API credentials.

Copy client_sample.json to client.json and fill in with the values you received.

## Running

  go build
  ./picturelife-experimental-uploader

Windows (and possibly OS X) firewalls may prompt to allow network access. To use the browser-based GUI, you should allow this.

Note: If you're running on Windows, need to change the root path value in web/assets/index.html to be something like "C:". 

The repository comes with a directory called 'data'. If you move the executable, you will need to create this directory.

## Configuration

Data files are stored in data the data directory and contain the database of locally indexed and uploaded files. They also contain configuration settings and access tokens.

The files are created the first time the API is connected to.

## CLI usage

By default, the uploader is controlled using a GUI webapp running at localhost:7111.

Basic functions are controllable via the CLI mode. Passing a path to a directory or file will upload that path. If you pass a directory, you can set the "-watch" flag to watch for changes in that directory and upload new or changed media.

The CLI is somewhat neglected.


