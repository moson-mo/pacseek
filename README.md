[![pacseek](https://img.shields.io/static/v1?label=pacseek&message=v1.3.3-1&color=1793d1&style=for-the-badge&logo=archlinux)](https://aur.archlinux.org/packages/pacseek/)
[![pacseek-bin](https://img.shields.io/static/v1?label=pacseek-bin&message=v1.3.3-1&color=1793d1&style=for-the-badge&logo=archlinux)](https://aur.archlinux.org/packages/pacseek-bin/)

[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/moson-mo/pacseek/Build)](https://github.com/moson-mo/pacseek/actions) 
[![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/pacseek)](https://goreportcard.com/report/github.com/moson-mo/pacseek)

<img src="/assets/pacseek.png" align="left" width="64"/>

# pacseek
## A terminal user interface for searching and installing Arch Linux packages

pacseek is terminal user interface which allows you to browse and search through the Arch Linux package databases as well as the Arch User Repository  

![pacseek](https://github.com/moson-mo/pacseek/blob/main/assets/pacseek_animation.gif?raw=true?inline=true)

Package installation / removal is done with an AUR helper. Make sure you have one installed.  
In the default configuration, [yay](https://github.com/Jguer/yay) is being used. You can change this in the settings.  

#### Libraries used

* [tview](https://github.com/rivo/tview) for the tui components
* [go-alpm](https://github.com/Jguer/go-alpm) for searching the package DB's
* [go-cache](https://github.com/patrickmn/go-cache) to cache search results and package data
* [goaurrpc](https://github.com/moson-mo/goaurrpc) self-hosted backend for the AUR REST API calls (to not stress the official aur.archlinux.org/rpc endpoint)

#### How to build / run / install

```
$ git clone https://github.com/moson-mo/pacseek.git
$ cd pacseek
$ go build .
$ ./pacseek
```

Binaries are available on the [Releases](https://github.com/moson-mo/pacseek/releases/) page.  
Also an [AUR package](https://aur.archlinux.org/packages/pacseek/) is available.

#### Navigation / Usage

You can either use the keyboard or mouse to navigate through the different components.  
While the search bar is focused, use the ENTER key to search for packages.  

With TAB you can navigate to the package list. Use the cursor keys to navigate within the list.  
To install/remove a package, press ENTER.  

The settings form can be opened with CTRL+S.  

More detailed information can be found in the [Wiki](https://github.com/moson-mo/pacseek/wiki/)


#### ToDo's

* Improve test code
* ~~Implement caching for package data to have less lookups (AUR)~~ - done
* ~~Add config options for disabling caching and setting the expiry time for cached entries~~ - done
* ~~Add docs / wiki~~ - done