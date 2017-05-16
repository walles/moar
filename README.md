Moar is a pager.  It's designed to just do the right thing without any
configuration:

![Moar displaying its own test suite](http://walles.github.io/moar/images/moar.png)

The intention is that Moar should work as a drop-in replacement for
[Less](http://www.greenwoodsoftware.com/less/). If you find that Moar
doesn't work that way,
[please report it](https://github.com/walles/moar/issues)!

Doing the right thing includes:

* **Syntax highlight** source code by default if
  [GNU Source-highlight](http://www.gnu.org/software/src-highlite/)
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

Getting the Latest Version
--------------------------
The latest version can be found at
<https://github.com/walles/moar>, or downloaded by doing
<pre>
  git clone https://github.com/walles/moar.git
</pre>

Installing
----------
Install it (in /usr/local/bin) by doing "rake install", or run
Moar directly from src/moar.rb. Do "rake help" to learn more about how
Rake can help you with installation.

Setting Moar as Your Default Pager
----------------------------------
Set it as your default pager by adding...

<pre>
  export PAGER=/usr/local/bin/moar
</pre>

... to your .bashrc.

Issues
------
Issues are tracked [here](https://github.com/walles/moar/issues), or
you can send questions to <johan.walles@gmail.com>.

Test Suite
----------
The test suite can be run by doing ./src/test.rb. If you have
[Rubocop](https://github.com/bbatsov/rubocop) installed it will be run
as part of the test suite.

Making a new Release
--------------------
First, to check version number of the most recent release:
* `git tag`

Then, to release the next one:
* `git tag --annotate <new version>`
* `git push --tags`

That's all there's to it!

TODO
----
* Read `source-highlight` output as a stream for startup performance reasons.
  This must work when `source-highlight` fails as well, and when it succeeds on
  an empty input file.

* Handle search hits to the right of the right screen edge. Searching
  forwards should move first right, then to the left edge and
  down. Searching backwards should move first left, then up and to the
  right edge (if needed for showing search hits).

* When skipping to the end, either while searching or when the user presses '>',
  try finding the end of the file for at most two seconds, then show wherever we
  are. Pressing '>' again or searching again should make another attempt until
  we're actually done.

* Make search work cross color boundaries. Currently, if you have a
  syntax highlighted line and search for something across a color
  change you won't get any match.

* Redefine 'g' without any prefix to prompt for which line to go
  to. This definition makes more sense to me than having to prefix 'g'
  to jump.

* Start at a certain line if run as "moar.rb file.txt:42"

* Enable home / end using home / end keys.

* Always print the name of the file being shown in the status field.

* Support viewing multiple files by pushing them in reverse order on
  the view stack.

* Auto generate the in-program help text to correctly correspond to
  the actual key bindings.

* Lazy load big / slow streams

* Add search line editing

* Try to find a newer Ruby version if needed for color support and
  exec() with that instead if available.

* Write "/ to search" somewhere in the status field

* Incremental search using ^s and ^r like in Emacs

* Gunzip input files with .gz extension before displaying them

* Warn but don't hang if we get an incomplete UTF-8 sequence from
  getch() in wide_getch().  Hanging won't be that much of a problem
  assuming users will press more keys if nothing happens, thus
  resolving the hang.

* Enable up / down using the mouse wheel.


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

* Incremental search using /

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

* Enable sideways scrolling using arrow keys.

* Warn about any unhandled keypresses during search.

* Enable displaying a man page
 * Arrow down through the whole file, then arrow up again
 * Page down through the whole file, then page up again
 * Search highlighting

* Make sure we get the line length right even with unicode characters
  present in the lines.  Verify by looking at where the truncation
  markers end up.

* Make sure we can search for unicode characters
 * Work around
 [the issue with getch not returning unicode chars](https://bugs.ruby-lang.org/issues/9094)

 * Work around
 [the issue with Regexp.quote() returning non-unicode strings](https://bugs.ruby-lang.org/issues/9096)

* Warn but don't crash if we get an invalid UTF-8 sequence from
  getch() in wide_getch().

* Make sure the LANG environment variable is printed if there are
warnings.

* Make sure some kind of platform information is printed if there are
warnings.

* Make sure the Ruby version is printed if there are warnings.

* Startup exceptions should be caught through the same reporting
  thingy as everything else.

* We must not crash on getting binary data.  Testcase: "moar.rb /bin/ls"

* Fix handling of TAB characters in the input

* Run rubocop as part of test.rb if installed and have the exit code
  reflect any issues.

* Make sure version information is printed if there are warnings.

* Make it possible to install system-wide using "rake install". Don't
  forget to fix the version number when doing this.

* Enable --help for help

* Enable --version for version information.

* Report command line errors, think about different command line
  requirements depending on whether we're piping input into moar.rb or
  listing input files on the command line.

  Command line formats we want to support:
  * moar.rb file.txt
  * moar.rb < file.txt

  Command line formats we *don't* want to support:
  * moar.rb file1.txt file2.txt
  * moar.rb file1.txt < file2.txt

* Enable 'h', '?' or F1 for help

* Print something nice on file-not-found.

* Test on Ubuntu

* Test on Ruby 1.8.something. We did, and due to missing UTF-8 support
  in Ruby 1.8 we just dropped support for it. Now we print an error
  message if Ruby < 1.9 is detected.

* Add info to the end of the --help output on how to set Moar to be
your default pager.

* Add licensing information (BSD)

* Enable source code highlighting by pre-filtering using GNU
  Source-highlight.

* Retain the search string when pressing / to search a second time.

* Exit search mode and cancel the search on ESC. Because that's what I
  feel like pressing.

* Exit search mode and cancel the search on ^G. For compatibility with
  Emacs.

* Make sure searching for an upper case unicode character turns on
  case sensitive search.

* Doing moar.rb on an arbitrary binary (like /bin/ls) should put all
  line-continuation markers at the rightmost column.  This really
  means our truncation code must work even with things like tabs and
  various control characters.

* Enable exiting using ^c (without restoring the screen).

* Enable pass-through operation unless $stdout.isatty()

* Accept numeric prefixes just like less. Implement for 'g', 'G' and
  SPACE to begin with.

* Exit search on pressing up / down / pageup / pagedown keys and
  scroll. I attempted to do that spontaneously, so it's probably a
  good idea.

* Searching for something above us should wrap the search.

* When pressing '/' to edit the search terms, find a hit and
  re-highlight.

* Make sure "git grep" output gets highlighted properly.
