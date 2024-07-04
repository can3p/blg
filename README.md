# blg - Command line blog client

blg is based on [cl-journal](https://github.com/can3p/cl-journal), but is written in
go in order to make contributing and distribution easier.

Supported services:

* [pcom](https://github.com/can3p/pcom)
* That's it!

Supported features:

- File based blog management
- blg supports both pushing to remote services and pulling from them
- Image upload is supported in case service implementation supports it
- Remote changes are merged into existing posts, including remote images
- During post download all the image paths are changed to the local one
  to allow to comfortably edit files locally

## Philosophy

`blg` tries to leave as much as possible to the discretion of a particular service
implementation. For example, No assumption is made about the post headers or
the format used by post body. The client only deals with all the surrounding
machinery like cli handling, storing files etc.

If you're curious or want to implement the support for your service, please
have a look on [interface definition](https://github.com/can3p/blg/blob/master/pkg/types/service.go)
and [pcom implementation](https://github.com/can3p/blg/blob/master/pkg/services/pcom/pcom.go)
to get the idea.

## Installation

### Install Script

Download `blg` and install into a local bin directory.

#### MacOS, Linux, WSL

Latest version:

```bash
curl -L https://raw.githubusercontent.com/can3p/blg/master/generated/install.sh | sh
```

Specific version:

```bash
curl -L https://raw.githubusercontent.com/can3p/blg/master/generated/install.sh | sh -s 0.0.4
```

The script will install the binary into `$HOME/bin` folder by default, you can override this by setting
`$CUSTOM_INSTALL` environment variable

### Manual download

Get the archive that fits your system from the [Releases](https://github.com/can3p/blg/releases) page and
extract the binary into a folder that is mentioned in your `$PATH` variable.

## Notes

The project has been scaffolded with the help of [kleiner](https://github.com/can3p/kleiner)
