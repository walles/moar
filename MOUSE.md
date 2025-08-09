# Mouse Scrolling vs Selecting and Copying

`moor` supports two mouse modes (using the `--mousemode` parameter):

- `scroll` makes `moor` process mouse events from your terminal, thus enabling mouse scrolling work,
but disabling the ability to select text with mouse in the usual way. Selecting text will require using your terminal's capability to bypass mouse protocol.
Most terminals support this capability, see [Selection workarounds for `scroll` mode](#mouse-selection-workarounds-for-scroll-mode) for details.
- `select` makes `moor` not process mouse events. This makes selecting and copying text work, but scrolling might not be possible, depending on your terminal and its configuration.
- `auto` uses `select` on terminals where we know it won't break scrolling, and
  `scroll` on all others. [The white list lives in the
  `terminalHasArrowKeysEmulation()` function in
  `screen.go`](https://github.com/walles/moor/blob/master/twin/screen.go).

The reason these tradeoffs exist is that if `moor` requests mouse events from the terminal,
it should process _all_ mouse events, including attempts to select text. This is the case with every console application.

However, some terminals can send "fake" arrow key presses to applications which _do not_ request processing mouse events.
This means that on those terminals, you will be better off using `--mousemode select` option, given that you also have this feature enabled (it's usually on by default).
With this setup, both scrolling and text selecting in the usual way will work.
To check whether this could work, simply run `moor` with option `--mousemode select` and see if scrolling still works.

## Mouse Selection Workarounds for `scroll` Mode

Most terminals implement a way to suppress mouse events capturing by applications, thus allowing you to select text even in
those applications which make use of the mouse. Usually this involves selecting with <kbd>Shift</kbd> being held. Often the
modifier key is configurable. Some other terminals allow setting options for specific types of mouse events to be reported.
While the table below attempts to list the default behaviours of some common terminals, you should consult
documentation of the one you're using to get detailed up-to-date information.

If your favorite terminal is missing, feel free to add it.

> :warning: With some of these, if you made incorrect selection you can cancel it either with an <kbd>Escape</kbd> key press or with a mouse
> click on text area. You will probably need to still hold the modifier key for this, as hitting <kbd>Escape</kbd> without it will likely exit `moor`.

| Terminal | Solution |
| -------- | -------- |
| Alacritty | [Use <kbd>Shift</kbd>](https://github.com/alacritty/alacritty/blob/a10fb8adc0857633be428d673ab8d36d2a90193a/extra/man/alacritty.5.scd#mouse) when selecting with mouse.<br>*Cred to @chrisgrieser for this tip.* |
| Contour | [Use <kbd>Shift</kbd>](https://github.com/contour-terminal/contour/blob/cf434eaae4b428228413039624231ad0a4e6839b/docs/configuration/advanced/mouse.md) when selecting with mouse.<br>*Cred to @postsolar for this tip.* |
| Foot | [Use <kbd>Shift</kbd>](https://codeberg.org/dnkl/foot/wiki#i-can-t-use-the-mouse-to-select-text) when selecting with mouse.<br>*Cred to @postsolar for this tip.* |
| Hyper | On macOS: Set `macOptionSelectionMode: 'force'` in your config file, then hold the Option Key <kbd>⌥</kbd> when selecting with mouse. |
| iTerm | Go to `Preferences / Profiles / Default / Terminal / ` and uncheck "Report mouse clicks & drags". |
| Terminal on macOS | On a laptop: Hold down the <kbd>fn</kbd> key when selecting with mouse. |
| kitty | [Use <kbd>Shift</kbd>](https://sw.kovidgoyal.net/kitty/overview/#mouse-features) or <kbd>Fn</kbd> when selecting with mouse.<br>*Cred to @PrayagS for this tip.* |
| Konsole | [Use <kbd>Shift</kbd>](https://github.com/KDE/konsole/blob/ffc48bfdadfb93fea09dcc9181b73738a4544a08/doc/manual/index.docbook#L242) when selecting with mouse.<br>*Cred to @cig0 for this tip.* |
| Terminator | Use <kbd>Shift</kbd> key when selecting with mouse.<br>*Cred to @felix-seifert for this tip.* |
| Tilix | Use <kbd>Shift</kbd> key when selecting with mouse.<br>*Cred to @Macr0Nerd for this tip.* |
| Warp | Go to `Preferences / Settings / Features / Terminal / ` and uncheck "Enable Mouse Reporting". |
| Windows | Use <kbd>Shift</kbd> key when selecting with mouse.<br>*Cred to @89z for this tip.* |

# `less`' screen initialization sequence

Recorded using [iTerm's _Automatically log session input to files_ feature](https://iterm2.com/documentation-preferences-profiles-session.html).

`less` is version 487 that comes with macOS 11.3 Big Sur.

All linebreaks are mine, added for readability. The `^M`s are not.

```
less /etc/passwd
^G<ESC>[30m<ESC>(B<ESC>[m^M
<ESC>[?1049h
<ESC>[?1h
<ESC>=^M
##
```

# `moor`'s screen initialization sequence

```
moor /etc/passwd /Users/johan/src/moor
^G<ESC>[30m<ESC>(B<ESC>[m^M
<ESC>[?1049h
<ESC>[?1006;1000h
<ESC>[?25l
<ESC>[1;1H
<ESC>[m<ESC>[2m  1 <ESC>[22m##
```

# Analysis of `less`

The line starting with `^G` is probably from from [`fish`](https://fishshell.com/) since it's the same for both `less` and `moor`.

`<ESC>[?1049h` switches to the Alternate Screen Buffer, [search here for `1 0 4 9`](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-The-Alternate-Screen-Buffer) for info.

Then `less` does `[?1h`, which apparently is [DECCKM Cursor Keys Mode, send ESC O A for cursor up](https://www.real-world-systems.com/docs/ANSIcode.html), followed by `=`, meaning [DECKPAM - Set keypad to applications mode (ESCape instead of digits)](https://www.real-world-systems.com/docs/ANSIcode.html).

**NOTE** that this means that `less` version 487 that comes with macOS 11.3 Big Sur doesn't even try to enable any mouse reporting, but relies on the terminal to convert scroll wheel events into arrow keypresses.

# Analysis of `moor`

Same as `less` up until the Alternate Screen Buffer is enabled.

`<ESC>[?1006;1000h` enables [SGR Mouse Mode and the X11 xterm mouse protocol (search for `1 0 0 0`)](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html).

`<ESC>[?25l` [hides the cursor](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html). **NOTE** Maybe we don't need this? It might be implicit when we enable the Alternate Screen Buffer.

`<ESC>[1;1H` [moves the cursor to the top left corner](<https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences>).

Then it's the first line with its line number in faint type.
