# Mouse Scrolling vs Copy / Paste

`moar` supports two mouse modes (using the `--mousemode` parameter):

- `scroll` makes scrolling work, but will require some gymnastics for copying
  text with your mouse, see below.
- `mark` makes copying text work, but on some terminals this will break scrolling.
- `auto` uses `mark` on terminals where we know it won't break scrolling, and
  `scroll` on all others. [The white list lives in the
  `mouseTrackingRecommended()` function in
  `screen.go`](https://github.com/walles/moar/blob/master/twin/screen.go).

The reason is that if `moar` requests mouse events from the terminal, that will
make the terminal not accept mark / copy any more.

If `moar` _doesn't_ request mouse events, then some terminals will send arrow
keys instead when you scroll your mouse wheel, making scrolling still work. Some
other terminals will send nothing, making scrolling not work.

`less --mouse` has the same problems.

## Text Marking Workarounds in `scroll` Mode

- **Alacritty**: Use use <kbd>shift</kbd> + mouse selection to make it work. Cred to @chrisgrieser for this tip.
- **Hyper** on macOS: Set `macOptionSelectionMode: 'force'` in your config file, then hold the Option Key <kbd>⌥</kbd> while marking
- **iTerm**: Preferences / Profiles / Default / Terminal / uncheck "Report mouse clicks & drags"
- macOS **Terminal** on a laptop: Hold down the <kbd>fn</kbd> key while marking with the mouse
- **kitty** on macOS: Holding <kbd>shift</kbd> or <kbd>fn</kbd> while marking. Cred to @PrayagS for this tip.
- **Konsole** Use use <kbd>shift</kbd> + mouse selection. Cred to @cig0 for this tip.
- **[Terminator](https://github.com/gnome-terminator/terminator)**: Use use <kbd>shift</kbd> + mouse selection to make it work. Cred to @felix-seifert for this tip.
- **[Tilix](https://gnunn1.github.io/tilix-web/)**: Use use <kbd>shift</kbd> + mouse selection. Cred to @Macr0Nerd for this tip.
- **[Warp](https://app.warp.dev)**: Preferences / Settings / Features / Terminal / uncheck "Enable Mouse Reporting"
- **Windows**: Use <kbd>Shift</kbd> to make a selection. If you click the wrong initial spot, you can clear the selection with <kbd>Esc</kbd>. Just be careful, if you hit <kbd>Esc</kbd> without a selection, the pager will exit. Cred to @89z for this one.

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

# `moar`'s screen initialization sequence

```
moar /etc/passwd /Users/johan/src/moar
^G<ESC>[30m<ESC>(B<ESC>[m^M
<ESC>[?1049h
<ESC>[?1006;1000h
<ESC>[?25l
<ESC>[1;1H
<ESC>[m<ESC>[2m  1 <ESC>[22m##
```

# Analysis of `less`

The line starting with `^G` is probably from from [`fish`](https://fishshell.com/) since it's the same for both `less` and `moar`.

`<ESC>[?1049h` switches to the Alternate Screen Buffer, [search here for `1 0 4 9`](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-The-Alternate-Screen-Buffer) for info.

Then `less` does `[?1h`, which apparently is [DECCKM Cursor Keys Mode, send ESC O A for cursor up](https://www.real-world-systems.com/docs/ANSIcode.html), followed by `=`, meaning [DECKPAM - Set keypad to applications mode (ESCape instead of digits)](https://www.real-world-systems.com/docs/ANSIcode.html).

**NOTE** that this means that `less` version 487 that comes with macOS 11.3 Big Sur doesn't even try to enable any mouse reporting, but relies on the terminal to convert scroll wheel events into arrow keypresses.

# Analysis of `moar`

Same as `less` up until the Alternate Screen Buffer is enabled.

`<ESC>[?1006;1000h` enables [SGR Mouse Mode and the X11 xterm mouse protocol (search for `1 0 0 0`)](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html).

`<ESC>[?25l` [hides the cursor](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html). **NOTE** Maybe we don't need this? It might be implicit when we enable the Alternate Screen Buffer.

`<ESC>[1;1H` [moves the cursor to the top left corner](<https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_(Control_Sequence_Introducer)_sequences>).

Then it's the first line with its line number in faint type.
