Moar is a pager.  It's designed to just do the right thing without any
configuration:

![Moar displaying its own source code](screenshot.png)

The intention is that Moar should work as a drop-in replacement for
[Less](http://www.greenwoodsoftware.com/less/). If you find that Moar
doesn't work that way,
[please report it](https://github.com/walles/moar/issues)!

Doing the right thing includes:

* **Syntax highlight** source code by default using
  [Chroma](https://github.com/alecthomas/chroma)
* **Search is incremental** / find-as-you-type just like in
  [Chrome](http://www.google.com/chrome) or
  [Emacs](http://www.gnu.org/software/emacs/)
* Search becomes case sensitive if you add any UPPER CASE characters
  to your search terms, just like in Emacs
* [Regexp](http://en.wikipedia.org/wiki/Regular_expression#Basic_concepts)
  search if your search string is a valid regexp
* Supports displaying ANSI color coded texts (like the output from
  `git diff` [| `riff`](https://github.com/walles/riff) for example)
* Supports UTF-8 input and output
* The position in the file is always shown

[For compatibility reasons](https://github.com/walles/moar/issues/14), `moar`
uses the formats declared in these environment variables when viewing man pages:

* `LESS_TERMCAP_md`: Bold
* `LESS_TERMCAP_us`: Underline

Moar is used as the default pager by:
* [`px` / `ptop`](https://github.com/walles/px)
* [`riff`](https://github.com/walles/riff)

Installing
----------

1. Download `moar` for your platform from
   <https://github.com/walles/moar/releases/latest>
1. `chmod a+x moar-*-*-*`
1. `sudo mv moar-*-*-* /usr/local/bin/moar`

And now you can just invoke `moar` from the prompt!

If a binary for your platform is not available, please
[file a ticket](https://github.com/walles/moar/releases) or contact
<johan.walles@gmail.com>.

Debian / Ubuntu
---------------

[A Request for Packaging is open](https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=944035),
please help!

Setting Moar as Your Default Pager
----------------------------------

Set it as your default pager by adding...

```bash
export PAGER=/usr/local/bin/moar
```

... to your `.bashrc`.

Issues
------

Issues are tracked [here](https://github.com/walles/moar/issues), or
you can send questions to <johan.walles@gmail.com>.

Embedding
---------

Here's one way to embed `moar` in your app:

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/walles/moar/m"
)

func main() {
	buf := new(bytes.Buffer)
	for range [99]struct{}{} {
		fmt.Fprintln(buf, "Moar")
	}

	err := m.NewPager(m.NewReaderFromStream("Moar", buf)).Page()
	if err != nil {
		// Handle paging problems
		panic(err)
	}
}
```

`m.Reader` can also be initialized using `NewReaderFromText()` or
`NewReaderFromFilename()`.

Developing
----------

You need the [go tools](https://golang.org/doc/install).

Run tests:

```bash
./test.sh
```

Build + run:

```bash
./moar.sh ...
```

Install (into `/usr/local/bin`) from source:

```bash
./install.sh
```

Making a new Release
--------------------

Make sure that [screenshot.png](screenshot.png) matches moar's current UI.
If it doesn't, scale a window to 81x16 characters and make a new one.

Execute `release.sh` and follow instructions.

TODO
----

* Searching for something above us should wrap the search.

* Enable exiting using ^c (without restoring the screen).

* Start at a certain line if run as "moar.rb file.txt:42"

* Redefine 'g' without any prefix to prompt for which line to go
  to. This definition makes more sense to me than having to prefix 'g'
  to jump.

* Handle search hits to the right of the right screen edge. Searching
  forwards should move first right, then to the left edge and
  down. Searching backwards should move first left, then up and to the
  right edge (if needed for showing search hits).

* Support viewing multiple files by pushing them in reverse order on
  the view stack.

* Incremental search using ^s and ^r like in Emacs

* Retain the search string when pressing / to search a second time.

Done
----

* Add '>' markers at the end of lines being cut because they are too long

* Doing moar on an arbitrary binary (like /bin/ls) should put all
  line-continuation markers at the rightmost column.  This really
  means our truncation code must work even with things like tabs and
  various control characters.

* Make sure search hits are highlighted even when we have to scroll right
  to see them

* Change out-of-file visualization to writing --- after the end of the
  file and leaving the rest of the screen blank.

* Exit search on pressing up / down / pageup / pagedown keys and
  scroll. I attempted to do that spontaneously, so it's probably a
  good idea.

* Remedy all FIXMEs in this README file

* Release the `go` version as the new `moar`, replacing the previous Ruby
  implementation

* Add licensing information (same as for the Ruby branch)

* Make sure "git grep" output gets highlighted properly.

* Handle all kinds of line endings.

* Make sure version information is printed if there are warnings.

* Add spinners while file is still loading

* Make `tail -f /dev/null` exit properly, fix
  <https://github.com/walles/moar/issues/7>.

* Showing unicode search hits should highlight the correct chars
