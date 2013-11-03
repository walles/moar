Moar is a pager.  It's designed to be easy to use and just do the
right thing without any configuration.

TODO (before trying to get others to use it)
--------------------------------------------
* Enable sideways scrolling using arrow keys.

* Make sure we can search for unicode characters

* Make sure we get the line length right even with unicode characters
  present in the lines.  Verify by looking at where the truncation
  markers end up.

* Make sure searching won't match part of a multi-byte unicode
  character.

* Handle search hits to the right of the right screen edge

* Enable 'h' or '?' for help

* Report command line errors, think about different command line
  requirements depending on whether we're piping input into moar.rb or
  listing input files on the command line.


TODO (bonus)
------------
* Start at a certain line if run as "moar.rb file.txt:42"

* Lazy load big / slow streams

* Add a search history

* Add search line editing

* Try to find a newer Ruby version if needed for color support and
  exec() with that instead if available.

* Enable displaying a man page
 * Arrow down through the whole file, then arrow up again
 * Page down through the whole file, then page up again
 * Search highlighting

* Make sure searching for an upper case unicode character turns on
  case sensitive search.

* Write "/ to search" somewhere in the status field

* Interactive search using ^s and ^r like in Emacs

* Enable filtered input, start with zcat as a filter

* Enable source code highlighting by pre-filtering using some
  highlighter.

* Enable exiting using ^c (doesn't restore screen).

* Just pass stuff through if stdout is not a terminal.

* Enable up / down using whatever less uses.

* Enable home / end using home / end keys.

* Enable up / down using the mouse wheel.

* Enable pass-through operation unless $stdout.isatty()

* Doing moar.rb on an arbitrary binary (like /bin/ls) should put all
  line-continuation markers at the rightmost column.  This really
  means our truncation code must work even with things like tabs and
  various control characters.


DONE
----
* Enable exiting using q (restores screen)

* Handle the terminal window getting resized.

* Print info line in inverse video

* Enable up / down using arrow keys.

* Prevent pressing down past the last line of the file.

* Enable out-of-file visualization with ~ like less.

* Enable up / down using page-up and page-down keys.

* Enable home / end using < and >.

* Enable file input.

* Enable continuous position display with everything we know (lines
  visible, percentages, like less).

* Enable stdin input.

* Truncate lines that are longer than the screen width

* Make sure we can print all the way into the rightmost column of the
  screen when truncating too long lines.  We should strip() lines
  before we print them and manually move the cursor to the next line
  after each.

* Handle all kinds of line endings.

* Handle files missing an ending newline.

* Handle hitting BACKSPACE in the search field

* Interactive search using /

* Typing backspace in the line editor when it's empty should make the
  line editor say "done".

* Change out-of-file visualization to writing --- after the end of the
  file and leaving the rest of the screen blank.

* Scroll down if we have no search hits on the current screen

* Wrap search if we have search hits above but not below

* Find next using n

* Highlight search hits using reverse video

* Make sure we can properly render all lines of /etc/php.ini.default
  without the bottom-of-the-screen prompt moving around.

* Find previous using N

* Indicate when we're wrapping the search while pressing n.

* Indicate when we're wrapping the search while pressing N.

* Highlight all matches while searching

* Scroll down one line on RETURN

* Print warnings to stderr after the run, for example if we aren't
  using color support because of a too-old version of Ruby.

* Make stdin input work even on newer (than 1.8) versions of
  Ruby. Apparently
  [this patch](http://svn.ruby-lang.org/cgi-bin/viewvc.cgi/trunk/io.c?r1=7641&r2=7649&diff_format=h)
  is the reason it doesn't
  work. [Reported to the Ruby issue tracker](https://bugs.ruby-lang.org/issues/9067),
  let's see how that goes.

* Enable displaying colorized output from "git diff"
 * Arrow down through the whole file, then arrow up again
 * Page down through the whole file, then page up again
 * Search highlighting

* Use the same algorithm for highlighting as for determining which
  lines match.

* Make the search case sensitive only if it contains any capital
  letters.

* Do a regexp search if the search term is a valid regexp, otherwise
  just use it as a substring.

* Make the search case sensitive only if it contains any capital
  letters, for both regexps and non-regexps.

* If we print warnings at the end, also print an URL where they can be
  reported.

* If we crash with a stacktrace, print an URL where it can be reported
