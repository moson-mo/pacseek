[![pacseek](https://img.shields.io/static/v1?label=pacseek&message=v1.6.3&color=1793d1&style=for-the-badge&logo=archlinux)](https://aur.archlinux.org/packages/pacseek/)
[![pacseek-bin](https://img.shields.io/static/v1?label=pacseek-bin&message=v1.6.3&color=1793d1&style=for-the-badge&logo=archlinux)](https://aur.archlinux.org/packages/pacseek-bin/)

[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/moson-mo/pacseek/Build)](https://github.com/moson-mo/pacseek/actions) 
[![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/pacseek)](https://goreportcard.com/report/github.com/moson-mo/pacseek)

<img src="/assets/pacseek.png" align="left" width="64"/>

# pacseek
## A terminal user interface for searching and installing Arch Linux packages

pacseek is terminal user interface which allows you to browse and search through the Arch Linux package databases as well as the Arch User Repository  

![pacseek](https://github.com/moson-mo/pacseek/blob/main/assets/pacseek_animation.gif?raw=true?inline=true)

Package installation / removal is done with an AUR helper. Make sure you have one installed.  
In the default configuration, [yay](https://github.com/Jguer/yay) is being used. You can change this in the settings.  

#### Features

* Search for packages in the Arch repositories and AUR
  * by: name / name & description
  * method: contains / starts-with
* Customizable commands for
  * Installing / Removing packages¹
  * Update all packages¹
  * Update repo packages
  * Show PKGBUILD³
* Adjustable appearance
  * Color schemes
  * Border styles
  * Component sizes / proportions
* ASCII mode for non unicode terminals
* Sortable search results by
  * Package name
  * Source
  * Installed state
  * Modified date
  * Popularity²
* Caching of
  * Search results
  * Package information
* Configurable AUR /rpc endpoint URL
* Display PKGBUILD file
* Search for upgrades / show list of upgradable packages⁴

¹ (requires an AUR helper. With the default config this is `yay`. You can change this in the settings)  
² (only applicable to AUR packages)  
³ (by default `curl` & `less` are used. Can be changed in the settings)  
⁴ (requires `fakeroot` to be installed)

#### Libraries used

* [tview](https://github.com/rivo/tview)
* [fuzzysearch](https://github.com/lithammer/fuzzysearch)
* [go-alpm](https://github.com/Jguer/go-alpm)
* [go-cache](https://github.com/patrickmn/go-cache)
* [goaurrpc](https://github.com/moson-mo/goaurrpc)
* [chroma](https://github.com/alecthomas/chroma)

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
To quit pacseek, you can use CTRL+Q, CTRL+C or ESC.

More detailed information can be found in the [Wiki](https://github.com/moson-mo/pacseek/wiki/)
