[![pacseek](https://img.shields.io/static/v1?label=pacseek&message=v1.8.3&color=1793d1&style=for-the-badge&logo=archlinux)](https://aur.archlinux.org/packages/pacseek/)
[![pacseek-bin](https://img.shields.io/static/v1?label=pacseek-bin&message=v1.8.3&color=1793d1&style=for-the-badge&logo=archlinux)](https://aur.archlinux.org/packages/pacseek-bin/)

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/moson-mo/pacseek/build.yml?branch=main)](https://github.com/moson-mo/pacseek/actions) 
[![Go Report Card](https://goreportcard.com/badge/github.com/moson-mo/pacseek)](https://goreportcard.com/report/github.com/moson-mo/pacseek)

<img src="/assets/pacseek.png" align="left" width="64"/>

# pacseek
## A terminal user interface for searching and installing Arch Linux packages

pacseek is terminal user interface which allows you to browse and search through the Arch Linux package databases as well as the Arch User Repository. Packages can be installed/uninstalled with the <kbd>ENTER</kbd> key.

![pacseek](https://github.com/moson-mo/pacseek/blob/main/assets/pacseek_animation.gif?raw=true?inline=true)

Package installation / removal is done with an AUR helper.   
In the default configuration, [yay](https://github.com/Jguer/yay) is being used.  
You can change this in the settings -> `Install command` / `AUR Install command`

There are some [examples for configuring other helpers](https://github.com/moson-mo/pacseek/wiki/Configuration#examples-for-other-aur-helpers) or even makepkg.
<br/>
<br/>

### Features

* Search for packages in the Arch repositories and AUR
  * by: name / name & description
  * method: contains / starts-with
* Auto-suggest (disabled by default)
* Customizable commands for
  * Installing / Removing packages¹
  * Update all packages¹
  * Update repo packages
  * Show PKGBUILD³
* Adjustable appearance
  * Color schemes
  * Border styles
  * Component sizes / proportions
  * Glyph styles
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
* Show a list of all installed packages
* News feed
  * Shown on upgrades screen
  * Feed URL(s) can be changed
  * Clicking on an item will `xdg-open` the URL to the article

¹ (By default, `yay` is being used to install/remove/upgrade packages. You can change this in the settings)  
² (only applicable to AUR packages)  
³ (by default `curl` & `less` are used. Can be changed in the settings)  
⁴ (requires `fakeroot` to be installed)
<br/>
<br/>

### Libraries used

* [tview](https://github.com/rivo/tview)
* [fuzzysearch](https://github.com/lithammer/fuzzysearch)
* [go-alpm](https://github.com/Jguer/go-alpm)
* [go-cache](https://github.com/patrickmn/go-cache)
* [goaurrpc](https://github.com/moson-mo/goaurrpc)
* [chroma](https://github.com/alecthomas/chroma)
* [gofeed](https://github.com/mmcdole/gofeed)
<br/>

### How to build / run / install

```
$ git clone https://github.com/moson-mo/pacseek.git
$ cd pacseek
$ go build .
$ ./pacseek
```

Binaries are available on the [Releases](https://github.com/moson-mo/pacseek/releases/) page.  
Also an [AUR package](https://aur.archlinux.org/packages/pacseek/) is available.
<br/>
<br/>

### Navigation / Usage

You can either use the keyboard or mouse to navigate through the different components.  
While the search bar is focused, use the <kbd>ENTER</kbd> key to search for packages.  

With <kbd>TAB</kbd> you can navigate to the package list. Use the cursor keys to navigate within the list.  
To install/remove a package, press <kbd>ENTER</kbd>.  

The settings form can be opened with <kbd>CTRL</kbd>+<kbd>S</kbd>.
To quit pacseek, you can use <kbd>CTRL</kbd>+<kbd>Q</kbd>, <kbd>CTRL</kbd>+<kbd>C</kbd> or <kbd>ESC</kbd>.
<br/>
<br/>

### Configuration

You change all configuration options from the settings screen (<kbd>CTRL</kbd>+<kbd>S</kbd>).  
The configuration file (json format) can be found at `~/.config/pacseek/config.json`
<br/>
<br/>

More detailed information regarding usage and configuration can be found in the [Wiki](https://github.com/moson-mo/pacseek/wiki/) or manpage: `man pacseek`
