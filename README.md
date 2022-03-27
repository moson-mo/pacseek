[![pacseek](https://img.shields.io/aur/version/pacseek?color=1793d1&label=pacseek&logo=arch-linux&style=for-the-badge)](https://aur.archlinux.org/packages/pacseek/)

[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/moson-mo/pacseek/Go)](https://github.com/moson-mo/pacseek/actions) 
[![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/pacseek)](https://goreportcard.com/report/github.com/moson-mo/pacseek)

# pacseek
## A terminal user interface for searching and installing Arch Linux packages

pacseek is terminal user interface which allows you to browse and search through the Arch Linux package databases as well as the Arch User Repository  

![pacseek](https://github.com/moson-mo/pacseek/blob/main/assets/pacseek_animation.gif?raw=true?inline=true)

Package installation / removal is done with an AUR helper. Make sure you have one installed.  
In the default configuration, [yay](https://github.com/Jguer/yay) is being used. You can change this in the settings.  

#### Libraries used

* [tview](https://github.com/rivo/tview) for the tui components
* [go-alpm](https://github.com/Jguer/go-alpm) for searching the package DB's
* [goaurrpc](https://github.com/moson-mo/goaurrpc) self-hosted backend for the AUR REST API calls (to not stress the official aur.archlinux.org/rpc endpoint)

#### How to build / run / install

```
$ git clone https://github.com/moson-mo/pacseek.git
$ cd pacseek
$ go build .
$ ./pacseek
```

Binaries are available on the [Releases](releases/) page.  
Also an [AUR package](https://aur.archlinux.org/packages/pacseek/) is available.

#### Navigation / Usage

You can either use the keyboard or mouse to navigate through the different components.  
While the search bar is focused, use the ENTER key to search for packages.  

With TAB you can navigate to the package list. Use the cursor keys to navigate within the list.  
To install/remove a package, press ENTER.  

The settings form can be opened with CTRL+S.  
