#!/usr/bin/env ruby
# -*- coding: utf-8 -*-

# Copyright (c) 2013, johan.walles@gmail.com
# All rights reserved.

require 'set'
require 'open3'
require 'curses'
require 'pathname'
require 'optparse'

MOAR_DIR = Pathname(__FILE__).realpath.dirname
VERSION = `cd #{MOAR_DIR} ; git describe`.strip

if RUBY_VERSION.to_f < 1.9
  if RUBY_PLATFORM =~ /darwin/
    $stderr.puts <<eos
ERROR: Moar requires at least OS X 10.9 Mavericks.

Or to be more precise, Moar requires Ruby 1.9, and OS X 10.9 Mavericks
is the first version of OS X to ship with a new enough Ruby.
eos
  else
    $stderr.puts <<eos
ERROR: Moar requires at least Ruby 1.9 and you are on Ruby #{RUBY_VERSION}.
eos
  end
  $stderr.puts <<eos

Ruby 1.9 brought:
* Support for different encodings. This is required for Moar to be
  able to display text with international characters in it.

* Support for the use_default_colors() NCurses function. This is
  required for Moar to be able to display colored text on the default
  terminal background.

If you have questions, please file a ticket at
https://github.com/walles/moar/issues or send a
question to johan.walles@gmail.com.
eos

  exit 1
end

# Editor for a line of text that can return its contents while
# editing. Needed for interactive search.
class LineEditor
  UPPER = /.*[[:upper:]].*/

  include Curses

  attr_reader :string
  attr_reader :warnings
  attr_reader :cursor_position

  def initialize(initial_string = '')
    @done = false
    @string = initial_string
    @cursor_position = 0
    @warnings = Set.new
  end

  def enter_char(char)
    case char
    when 10  # 10=RETURN on a Powerbook
      @done = true
    when 127 # 127=BACKSPACE on a Powerbook
      @string = @string[0..-2]
      @cursor_position -= 1

      if @cursor_position < 0 && @string.empty?
        @done = true
      end
    else
      begin
        @string << char.chr
        @cursor_position += 1
      rescue RangeError
        # These errors intentionally ignored; it's better to do
        # nothing than to crash if we get an unexpected / unsupported
        # keypress.
        @warnings << "Unhandled key while searching: #{char}"
      end
    end
    @cursor_position = [@cursor_position, 0].min
    @cursor_position = [@cursor_position, @string.length].max
  end

  def regexp
    options = Regexp::FIXEDENCODING | Regexp::IGNORECASE
    options = Regexp::FIXEDENCODING if @string =~ UPPER

    begin
      return Regexp.new(@string, options)
    rescue RegexpError
      # The force_encoding() thing on the next line is a workaround for
      # https://bugs.ruby-lang.org/issues/9096
      return Regexp.new(Regexp.quote(@string).force_encoding(Encoding::UTF_8),
                        options)
    end
  end

  def done?
    return @done
  end

  def empty?
    return @string.empty?
  end
end

# A string containing ANSI escape codes
class AnsiString
  ESC = 27.chr
  TAB = 9.chr
  CONTROLCODE = /[#{0.chr}-#{8.chr}#{10.chr}-#{26.chr}#{28.chr}-#{31.chr}]/
  ANSICODE = /#{ESC}\[([0-9;]*m)/
  MANPAGECODE = /[^\b][\b][^\b]([\b][^\b])?/

  BOLD = "#{ESC}[1m"
  NONBOLD = "#{ESC}[22m"
  UNDERLINE = "#{ESC}[4m"
  NONUNDERLINE = "#{ESC}[24m"

  def initialize(string)
    string = to_utf8(string)
    string = manpage_to_ansi(string)
    string = scrub(string)
    string = resolve_tabs(string)
    @string = string
  end

  def resolve_tabs(string)
    return string unless string.index(TAB)
    resolved = ''

    offset = 0
    while true
      tabindex = string.index(TAB, offset)
      return resolved + string[offset..-1] unless tabindex

      resolved += string[offset..(tabindex - 1)] unless offset == tabindex
      offset = tabindex + 1

      n_spaces = 8 - (resolved.length % 8)
      n_spaces = 8 if n_spaces == 0
      resolved += ' ' * n_spaces
    end
  end

  # Replace control codes with "^X" where X is representative for the
  # actual control code replaced.
  def scrub(string)
    return string.gsub(CONTROLCODE) do |match|
      "^#{(match[0].ord + 64).chr}"
    end
  end

  def to_utf8(string)
    string.force_encoding(Encoding::ASCII_8BIT) unless string.valid_encoding?
    return string.encode(Encoding::UTF_8, :undef => :replace)
  end

  def manpage_to_ansi(string)
    return_me = ''

    is_bold = false
    is_underline = false
    while true
      (head, match, tail) = string.partition(MANPAGECODE)
      break if match.empty?

      unless head.empty?
        if is_underline
          return_me += NONUNDERLINE
          is_underline = false
        end

        if is_bold
          return_me += NONBOLD
          is_bold = false
        end

        return_me += head
      end

      char = match[-1]
      want_bold = false
      want_underline = false
      decorations = [match[0]]
      decorations << match[2] if match.length == 5
      decorations.each do |decoration|
        case decoration
        when char
          want_bold = true
        when '_'
          want_underline = true
        else
          # FIXME: Warn about this case
        end
      end

      if want_bold && !is_bold
        return_me += BOLD
        is_bold = true
      end

      if want_underline && !is_underline
        return_me += UNDERLINE
        is_underline = true
      end

      if is_underline && !want_underline
        return_me += NONUNDERLINE
        is_underline = false
      end

      if is_bold && !want_bold
        return_me += NONBOLD
        is_bold = false
      end

      return_me += char

      string = tail
    end

    return_me += NONUNDERLINE if is_underline
    return_me += NONBOLD if is_bold
    return_me += string

    return return_me
  end

  # Input: A string
  #
  # The string is divided into pairs of ansi escape codes and the text
  # following each of them.  The pairs are passed one by one into the
  # block.
  def tokenize(&block)
    last_match = nil
    string = @string
    while true
      (head, match, tail) = string.partition(ANSICODE)
      break if match.empty?
      match = Regexp.last_match[1]

      if last_match || !head.empty?
        block.call(last_match, head)
      end
      last_match = match

      string = tail
    end
    block.call(last_match, string)
  end

  # Input:
  #  A base string, optionally containing ANSI escape codes to put
  #  highlights in
  #
  #  Something to highlight
  #
  # Return:
  #  The base string with the highlights highlighted in reverse video
  def highlight(highlight)
    return_me = ''

    tokenize do |code, text|
      return_me += "#{ESC}[#{code}" if code
      left = text

      while true
        (head, match, tail) = left.partition(highlight)
        break if match.empty?

        return_me += head
        return_me += "#{ESC}[7m"  # Reverse video
        return_me += match
        return_me += "#{ESC}[27m" # Non-reversed video

        left = tail
      end

      return_me += left
    end

    return AnsiString.new(return_me)
  end

  # Return a substring starting at index start_index
  def substring(start_index)
    return self if start_index == 0

    string = ''
    seen = 0
    tokenize do |code, text|
      string += "#{ESC}[#{code}" if code

      if seen < start_index
        start_index_in_current_text = start_index - seen
        if start_index_in_current_text >= 0
          subtext = text[start_index_in_current_text..-1]
          string += subtext if subtext
        end
      else
        string += text
      end

      seen += text.length
    end

    return AnsiString.new(string)
  end

  def include?(search_term)
    tokenize do |code, text|
      return true if text.index(search_term)
    end

    return false
  end

  def to_str
    return @string
  end
end

# Displays the contents of a Moar instance
class Terminal
  include Curses

  attr_reader :warnings

  def colorized?
    if @colorized.nil?
      @colorized = Curses.respond_to?('use_default_colors')
    end
    return @colorized
  end

  def initialize(testing = false)
    @warnings = Set.new

    return if testing

    init_screen
    if colorized?
      start_color
      use_default_colors
    else
      @warnings <<
        'Need a newer Ruby version for color support, ' +
        "currently running Ruby #{RUBY_VERSION}"
    end

    noecho
    stdscr.keypad(true)
    cbreak

    @color_pairs = {}
    @next_color_pair_number = 1
  end

  def get_color_pair(foreground, background)
    pair = @color_pairs[[foreground, background]]
    unless pair
      pair = @next_color_pair_number
      @next_color_pair_number += 1

      init_pair(pair, foreground, background)
      @color_pairs[[foreground, background]] = pair
    end

    return color_pair(pair)
  end

  def close
    close_screen
  end

  # Return the number of lines of content this terminal can show.
  # This is generally the number of actual screen lines minus one for
  # the status line.
  def lines
    return super - 1
  end

  # This method is a workaround for
  # https://bugs.ruby-lang.org/issues/9094
  def wide_getch(*test_input)
    testing = !test_input.empty?
    byte = testing ? test_input.shift : getch

    return nil if byte.nil?

    # If it's already a character we assume it's fine
    return byte unless byte.is_a? Fixnum

    # Not within a byte = ncurses special, return unmodified
    return byte if byte < 0
    return byte if byte > 255

    # ASCII
    if byte <= 127
      if byte >= 32 && byte <= 126
        # For Ruby 1.8 compatibility
        byte = byte.chr
      end

      return byte
    end

    # Find the number of bytes in the sequence
    size = nil
    if byte & 0b1110_0000 == 0b1100_0000
      size = 2
    elsif byte & 0b1111_0000 == 0b1110_0000
      size = 3
    elsif byte & 0b1111_1000 == 0b1111_0000
      size = 4
    else
      @warnings <<
        "Invalid UTF-8 start byte #{byte} from keyboard"
      return byte.chr
    end

    bytes = [byte]
    (size - 1).times do
      bytes << (testing ? test_input.shift : getch)

      unless bytes[-1] & 0b1100_0000 == 0b1000_0000
        @warnings <<
          sprintf('Invalid UTF-8 sequence [%s] from keyboard, ' +
                  'LANG=%s',
                  bytes.map { |b| sprintf('0x%02x', b) }.join(', '),
                  ENV['LANG'])
        return bytes[0].chr
      end
    end

    return bytes.pack('C*').force_encoding(Encoding::UTF_8)
  end

  def add_search_status(moar)
    attrset(A_NORMAL)
    status = "/#{moar.search_editor.string}"
    addstr(status)
  end

  def add_notfound_status(moar)
    status = "Not found: #{moar.search_editor.string}"
    attrset(A_REVERSE)
    addstr(status)
  end

  def add_line(moar, screen_line, line)
    attrset(A_NORMAL)
    setpos(screen_line, 0)
    clrtoeol

    unless moar.search_editor.empty?
      line = line.highlight(moar.search_editor.regexp)
    end

    # Higlight search matches
    printed_chars = 0
    foreground = -1
    background = -1
    old_foreground = -1
    old_background = -1
    line.substring(moar.first_column).tokenize do |code, text|
      unless code.nil?
        unless code.end_with? 'm'
          @warnings << "Unsupported ANSI code \"#{code}\""
        end

        code[0..-2].split(';').each do |csi_code|
          csi_code = csi_code.to_i unless csi_code.empty?
          case csi_code
          when ''
            attrset(A_NORMAL)
            foreground = -1
            background = -1
          when 1
            attron(A_BOLD)
          when 4
            attron(A_UNDERLINE)
          when 7
            attron(A_REVERSE)
          when 22
            attroff(A_BOLD)
          when 24
            attroff(A_UNDERLINE)
          when 27
            attroff(A_REVERSE)
          when 30
            foreground = COLOR_BLACK
          when 31
            foreground = COLOR_RED
          when 32
            foreground = COLOR_GREEN
          when 33
            foreground = COLOR_YELLOW
          when 34
            foreground = COLOR_BLUE
          when 35
            foreground = COLOR_MAGENTA
          when 36
            foreground = COLOR_CYAN
          when 37
            foreground = COLOR_WHITE
          else
            @warnings << "Unsupported ANSI CSI code \"#{csi_code}\""
          end
        end
      end

      if colorized?
        if foreground != old_foreground || background != old_background
          attron(get_color_pair(foreground, background))

          old_foreground = foreground
          old_background = background
        end
      end

      addstr(text)

      printed_chars += text.length
      break if printed_chars > cols
    end

    # Print a continuation character if we've printed outside the
    # window
    if printed_chars > cols
      setpos(screen_line, cols - 1)
      attrset(A_REVERSE)
      addstr('>')
    end

    if moar.first_column > 0
      setpos(screen_line, 0)
      attrset(A_REVERSE)
      addstr('<')
    end
  end

  def add_view_status(moar)
    status = nil
    if moar.lines.size > 0
      status = "Lines #{moar.first_line + 1}-"

      status += "#{moar.last_line + 1}"

      status += "/#{moar.lines.size}"

      percent_displayed =
        (100 * (moar.last_line + 1) / moar.lines.size).floor
      status += " #{percent_displayed}%"
    else
      status = 'Lines 0-0/0'
    end

    if moar.first_column > 0
      status += "  Column #{moar.first_column}"
    end

    status += "  Last key=#{moar.last_key}"

    attrset(A_REVERSE)
    addstr(status)
  end

  def draw_screen(moar)
    screen_line = 0

    # Draw lines
    (moar.first_line..moar.last_line).each do |line_number|
      add_line(moar, screen_line, moar.lines[line_number])
      screen_line += 1
    end

    # Draw filling after EOF
    if screen_line < lines
      setpos(screen_line, 0)
      clrtoeol
      attrset(A_REVERSE)
      addstr('---')
      screen_line += 1
    end

    while screen_line < lines
      setpos(screen_line, 0)
      clrtoeol
      screen_line += 1
    end

    # Draw status line
    setpos(lines, 0)
    clrtoeol
    case moar.mode
    when :viewing
      add_view_status(moar)
    when :searching
      add_search_status(moar)
    when :notfound
      add_notfound_status(moar)
    else
      abort("ERROR: Unsupported mode of operation <#{@mode}>")
    end

    refresh
  end
end

# The pager logic is in this class; and it's displayed by the Terminal
# class
class Moar
  BUGURL = 'https://github.com/walles/moar/issues'

  attr_reader :lines
  attr_reader :search_editor
  attr_reader :mode
  attr_reader :last_key
  attr_reader :first_column

  def add_line(line)
    lines << AnsiString.new(line.rstrip)
  rescue => e
    return if @unhandled_line_warning

    bytes_dump =
      line.unpack('C*').map { |byte| sprintf('%3d', byte) }.join(',')

    @unhandled_line_warning =
      sprintf("Ignoring unhandled line: %s:\n[\n %s\n]",
              e.message,
              bytes_dump)
  end

  def initialize(file, terminal = Terminal.new)
    @view_stack = []
    @unhandled_line_warning = nil
    @search_editor = LineEditor.new
    @terminal = terminal
    @first_line = 0
    @first_column = 0
    @lines = nil
    push_view(file)
    @last_key = 0
    @done = false

    # Mode can be :viewing and :searching
    @mode = :viewing
  ensure
    @mode == :viewing || @terminal.respond_to?('close') && @terminal.close
  end

  def first_line
    # @first_line must not be closer than lines-2 from the end
    max_first_line = @lines.size - @terminal.lines
    @first_line = [@first_line, max_first_line].min

    # @first_line cannot be negative
    @first_line = [0, @first_line].max

    return @first_line
  end

  # Compute the last line given a first line
  def last_line(my_first_line = nil)
    my_first_line = first_line unless my_first_line

    # my_first_line must not be closer than lines-2 from the end
    max_first_line = @lines.size - @terminal.lines
    my_first_line = [my_first_line, max_first_line].min

    # my_first_line cannot be negative
    my_first_line = [0, my_first_line].max

    return_me = my_first_line + @terminal.lines - 1
    return_me = [@lines.size - 1, return_me].min

    return return_me
  end

  def last_line=(new_last_line)
    @first_line = new_last_line - @terminal.lines + 1
  end

  def find_next(direction = :forwards)
    return if @search_editor.empty?

    hit = full_search(@search_editor.regexp, direction)
    if hit
      show_line(hit)
      @mode = :viewing
    else
      @mode = :notfound
    end
  end

  def push_view(text)
    @view_stack << [@lines, first_line] unless @lines.nil?

    if text.is_a? String
      @lines = []
      text.lines.each do |line|
        add_line(line)
      end
    elsif text.is_a? Array
      # We got an array, used for unit testing
      @lines = []
      text.each do |line|
        add_line(line)
      end
    else
      @lines = []
      text.each_line do |line|
        add_line(line)
      end
    end
  end

  def pop_view
    view = @view_stack.pop
    if view.nil?
      @done = true
      return
    end

    @lines, @first_line = view
  end

  def helptext
    return <<eos
Welcome to Moar, the nice pager!

Quitting
--------
* Press 'q' to quit

Moving around
-------------
* Arrow keys
* PageUp and PageDown
* Home and End
* < to go to the start of the document
* > to go to the end of the document
* RETURN moves down one line
* SPACE moves down a page

Searching
---------
* Type / to start searching, then type what you want to find
* Type RETURN to stop searching
* Find next by typing 'n'
* Find previous by typing SHIFT-N
* Search is case sensitive if it contains any UPPER CASE CHARACTERS
* Search is interpreted as a regexp if it is a valid one

Reporting bugs
--------------
File issues at https://github.com/walles/moar/issues, or post
questions to johan.walles@gmail.com.

Installing Moar as your default pager
-------------------------------------
Put the following line in your .bashrc or .bash_profile:
  export PAGER=#{Pathname(__FILE__).realpath}
eos
  end

  def handle_view_keypress(key)
    case key
    when 'q'
      pop_view
    when 'h', '?', Curses::Key::F1
      push_view(helptext) if @view_stack.empty?
      @mode = :viewing
    when '/'
      @mode = :searching
      @search_editor = LineEditor.new
    when 'n'
      find_next(:forwards)
    when 'N'
      find_next(:backwards)
    when Curses::Key::RESIZE
      # Do nothing; draw_screen() will be called anyway between all
      # keypresses
    when Curses::Key::DOWN, 10  # 10=RETURN on a Powebook
      @first_line += 1
      @mode = :viewing
    when Curses::Key::UP
      @first_line -= 1
      @mode = :viewing
    when Curses::Key::RIGHT
      @first_column += 16
      @mode = :viewing
    when Curses::Key::LEFT
      @first_column -= 16
      @first_column = 0 if @first_column < 0
      @mode = :viewing
    when Curses::Key::NPAGE, ' '[0]
      @first_line = last_line + 1
      @mode = :viewing
    when Curses::Key::PPAGE
      self.last_line = first_line - 1
      @mode = :viewing
    when '<'
      @first_line = 0
      @first_column = 0
      @mode = :viewing
    when '>'
      @first_line = @lines.size
      @first_column = 0
      @mode = :viewing
    end
  end

  def handle_search_keypress(key)
    @search_editor.enter_char(key)
    if full_search_required?
      hit = full_search(@search_editor.regexp)
      show_line(hit) if hit
    end
    if @search_editor.done?
      @mode = :viewing
    end
  end

  # Get a certain line number on-screen
  def show_line(line_number)
    new_first_line = line_number

    # Move at least one screen away from where we were
    if new_first_line < first_line
      # Moving up
      if last_line(new_first_line) >= first_line
        self.last_line = first_line - 1
        return
      end
    end

    if new_first_line > last_line
      # Moving down
      new_first_line =
        [new_first_line, last_line + 1].max
    end

    @first_line = new_first_line
  end

  # Search the given line number range.
  #
  # Returns the line number of the first hit, or nil if nothing was
  # found.
  def search_range(first, last, find_me)
    line_numbers = first.upto(last)
    if last < first
      line_numbers = first.downto(last)
    end

    line_numbers.each do |line_number|
      if @lines[line_number].include?(find_me)
        return line_number
      end
    end

    return nil
  end

  # Search the full document and return the line number of the first
  # hit, or nil if nothing was found
  def full_search(find_me, direction = :forwards)
    from = nil
    to = nil

    if direction == :forwards
      to = @lines.size - 1
      if @mode == :notfound
        from = 0
      else
        from = last_line + 1
        if from >= @lines.size
          from = @lines.size - 1
        end
      end
    else
      to = 0
      if @mode == :notfound
        from = @lines.size - 1
      else
        from = first_line - 1
        if from < 0
          from = 0
        end
      end
    end

    return search_range(from, to, find_me)
  end

  def full_search_required?
    return false if @search_editor.empty?
    return !search_range(first_line, last_line, @search_editor.regexp)
  end

  def run
    until @done
      @terminal.draw_screen(self)

      key = @terminal.wide_getch
      case @mode
      when :viewing, :notfound
        handle_view_keypress(key)
      when :searching
        handle_search_keypress(key)
      else
        abort("ERROR: Unsupported mode of operation <#{@mode}>")
      end

      @last_key = key
    end
  end

  def close
    @terminal.close unless @terminal.nil?
  end

  def warnings
    return_me = Set.new
    return_me << @unhandled_line_warning if @unhandled_line_warning
    return_me.merge(@terminal.warnings)
    return_me.merge(@search_editor.warnings)

    return return_me
  end
end

# Command line options parser
class MoarOptions
  def initialize(options = ARGV)
    @version = false
    @help = false
    @error = nil
    @highlight = true
    parser.parse!(options)

    fail 'Only one file can be shown' if options.length > 1
    @file = options[0] unless options.empty?

    if @file && !File.exist?(@file)
      fail "File not found: #{@file}"
    end

    if @file && !File.file?(Pathname(@file).realpath)
      fail "Not a file: #{@file}"
    end
  rescue => e
    @file = nil
    @error = e.message
  end

  def help
    message = parser.help
    pager = ENV['PAGER']
    if pager && !File.exist?(pager)
      pager = nil
    end
    pager = Pathname(pager).realpath unless pager.nil?
    unless pager == Pathname(__FILE__).realpath
      message += <<eos

To make Moar your default pager, put the following line in
your .bashrc or .bash_profile and it will be default in all
new terminal windows:
  export PAGER=#{Pathname(__FILE__).realpath}
eos
    end

    if `which source-highlight`.empty?
      message += <<eos

To enable syntax highlighting when viewing source code, install GNU
Source-highlight (http://www.gnu.org/software/src-highlite).
eos
    end

    message += <<eos

Report issues at https://github.com/walles/moar/issues, or post
questions to johan.walles@gmail.com.
eos

    return message
  end

  def parser
    return OptionParser.new do |parser|
      parser.banner =
        "Usage:\n" +
        "  moar [options] <file>\n" +
        "  ... | moar\n" +
        "  moar < file\n\n"

      parser.on('-v', '--version', 'Show version information') do
        @version = true
      end

      parser.on('-h', '--help', 'Show this help') do
        @help = true
      end

      parser.on('--no-highlight', 'Don\'t highlight source code') do
        @highlight = false
      end
    end
  end
  private :parser

  def version?
    return false if @error
    return @version
  end

  def help?
    return false if @error
    return @help
  end

  def highlight?
    return @highlight
  end

  def error
    return @error
  end

  def file
    return @file
  end

  def print_help_and_exit(problem = nil)
    stream = (problem.nil? ? $stdout : $stderr)
    unless problem.nil?
      stream.puts problem
      stream.puts
    end
    stream.puts help

    exitcode = (problem.nil? ? 0 : 1)
    exit exitcode
  end
end

# Attempt to highlight the file using GNU Source-highlight
def highlight(file)
  lines = nil
  exitcode = nil
  Open3.popen3('source-highlight', '--out-format=esc',
               '-i', file,
               '-o', 'STDOUT') do |stdin, stdout, stderr, wait_thr|
    lines = stdout.readlines
    exitcode = wait_thr.value
  end
  return lines if exitcode.success?
  return File.open(file, 'r')
rescue
  # Source-highlight not installed
  return File.open(file, 'r')
end

moar = nil
crash = nil
begin
  if __FILE__ != $PROGRAM_NAME
    # We're being required, probably due to unit testing.
    # Do nothing.
  elsif $stdin.isatty
    options = MoarOptions.new
    if options.error
      options.print_help_and_exit('ERROR: ' + options.error)
    end
    if options.help?
      options.print_help_and_exit
    end
    if options.version?
      puts "Moar version #{VERSION}, see also https://github.com/walles/moar"
      exit 0
    end

    unless options.file
      options.print_help_and_exit('ERROR: Please add a file to view')
    end

    moar = nil
    if options.highlight?
      moar = Moar.new(highlight(options.file))
    else
      moar = Moar.new(File.open(options.file, 'r'))
    end
    moar.run
  else
    unless ARGV.empty?
      MoarOptions.new([]).print_help_and_exit 'ERROR: ' +
        "No options supported while reading from a pipe, got #{ARGV}"
    end

    # Switch around some fds to enable us to read the former stdin and
    # curses to read the "real" stdin.
    stream = $stdin.clone
    $stdin.reopen(IO.new(1, 'r+'))
    moar = Moar.new(stream)
    moar.run
  end
rescue => e
  crash = e
ensure
  moar.close unless moar.nil?

  warnings = Set.new
  warnings.merge(moar.warnings) if moar

  warnings.sort.each do |warning|
    $stderr.puts "WARNING: #{warning}"
  end

  if crash
    $stderr.puts unless warnings.empty?
    $stderr.puts("#{crash.class}: #{crash.message}")
    $stderr.puts('  ' + crash.backtrace.join("\n  "))
  end

  if crash || !warnings.empty?
    $stderr.puts
    $stderr.puts "Moar version: #{VERSION}"
    $stderr.puts "Ruby version: #{RUBY_VERSION}"
    $stderr.puts "Ruby platform: #{RUBY_PLATFORM}"
    $stderr.puts "LANG=<#{ENV['LANG']}>"
    ENV.each do |var, value|
      $stderr.puts "#{var}=<#{value}>" if var.start_with? 'LC_'
    end
    $stderr.puts
    $stderr.puts "Please report issues to #{Moar::BUGURL}"
  end

  exit 1 if crash
end
