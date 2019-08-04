Moar is a pager.  It's designed to just do the right thing without any
configuration:

![Moar displaying its own source code](screenshot.png)

The intention is that Moar should work as a drop-in replacement for
[Less](http://www.greenwoodsoftware.com/less/). If you find that Moar
doesn't work that way,
[please report it](https://github.com/walles/moar/issues)!

Doing the right thing includes:

* **Syntax highlight** source code by default if
  [Highlight](http://www.andre-simon.de/doku/highlight/en/highlight.php)
  is installed.
* **Search is incremental** / find-as-you-type just like in
  [Chrome](http://www.google.com/chrome) or
  [Emacs](http://www.gnu.org/software/emacs/)
* Search becomes case sensitive if you add any UPPER CASE characters
  to your search terms, just like in Emacs
* [Regexp](http://en.wikipedia.org/wiki/Regular_expression#Basic_concepts)
  search if your search string is a valid regexp
* Supports displaying ANSI color coded texts (like the output from
  "git diff" for example)
* Supports UTF-8 input and output
* The position in the file is always shown

Installing
----------

FIXME: End-user install instructions for the Go-built product

...

And now you can just invoke `moar` from the prompt!

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

Developing
----------

First, install [Highlight](http://www.andre-simon.de/zip/download.php),
otherwise the test suite won't pass:

* On macOS: [`brew install highlight`](https://brew.sh/)
* On Ubuntu: [`sudo apt-get install highlight`](https://packages.ubuntu.com/search?suite=all&searchon=names&keywords=highlight)
* Elsewhere, follow [instructions](http://www.andre-simon.de/zip/download.php)

Also, you need the [go tools](https://golang.org/doc/install).

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

Execute `release.sh` and follow instructions.

TODO
----

* Remedy all FIXMEs in this README file

* Release the `go` version as the new `moar`, replacing the previous Ruby
  implementation

* Searching for something above us should wrap the search.

* Handle all kinds of line endings.

* Enable exiting using ^c (without restoring the screen).

* Start at a certain line if run as "moar.rb file.txt:42"

* Add licensing information (BSD)

* Make sure version information is printed if there are warnings.

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

* Make sure "git grep" output gets highlighted properly.

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
