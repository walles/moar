#!/usr/bin/env ruby

require 'set'
require 'curses'

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
        @warnings << "WARNING: Unhandled key while searching: #{char}"
      end
    end
    @cursor_position = [@cursor_position, 0].min
    @cursor_position = [@cursor_position, @string.length].max
  end

  def regexp
    options = Regexp::IGNORECASE
    options = nil if @string =~ UPPER

    begin
      return Regexp.new(@string, options)
    rescue RegexpError
      return Regexp.new(Regexp.quote(@string), options)
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
  ANSICODE = /#{ESC}\[([0-9;]*m)/
  MANPAGECODE = /[^\b][\b][^\b]([\b][^\b])?/

  BOLD = "#{ESC}[1m"
  NONBOLD = "#{ESC}[22m"
  UNDERLINE = "#{ESC}[4m"
  NONUNDERLINE = "#{ESC}[24m"

  def initialize(string)
    @string = manpage_to_ansi(string)
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

  def initialize
    @warnings = Set.new

    init_screen
    if colorized?
      start_color
      use_default_colors
    else
      @warnings <<
        'WARNING: Need a newer Ruby version for color support, ' +
        "currently running Ruby #{RUBY_VERSION}"
    end

    noecho
    stdscr.keypad(true)
    crmode

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

  def getch
    super
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
      case code
      when nil
        # This case intentionally left blank
      when 'm'
        attrset(A_NORMAL)
        foreground = -1
        background = -1
      when '1m'
        attron(A_BOLD)
      when '4m'
        attron(A_UNDERLINE)
      when '7m'
        attron(A_REVERSE)
      when '22m'
        attroff(A_BOLD)
      when '24m'
        attroff(A_UNDERLINE)
      when '27m'
        attroff(A_REVERSE)
      when '30m'
        foreground = COLOR_BLACK
      when '31m'
        foreground = COLOR_RED
      when '32m'
        foreground = COLOR_GREEN
      when '33m'
        foreground = COLOR_YELLOW
      when '34m'
        foreground = COLOR_BLUE
      when '35m'
        foreground = COLOR_MAGENTA
      when '36m'
        foreground = COLOR_CYAN
      when '37m'
        foreground = COLOR_WHITE
      else
        @warnings << "WARNING: Unsupported ANSI code \"#{code}\""
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

  def initialize(file, terminal = Terminal.new)
    @search_editor = LineEditor.new
    @terminal = terminal
    @first_line = 0
    @first_column = 0
    if file.respond_to? '[]'
      # We got an array, used for unit testing
      @lines = []
      file.each do |line|
        lines << AnsiString.new(line.rstrip)
      end
    else
      @lines = []
      file.each_line do |line|
        lines << AnsiString.new(line.rstrip)
      end
    end
    @last_key = 0
    @done = false

    # Mode can be :viewing and :searching
    @mode = :viewing
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

  def handle_view_keypress(key)
    # For Ruby 1.8 compatibility
    begin
      key = key.chr unless key.nil?
    rescue => e
      # RangeErrors can happen for non-letter keys and are
      # intentionally ignored
      raise unless e.is_a? RangeError
    end

    case key
    when 'q'
      @done = true
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
    when Curses::Key::DOWN, 10  # 10=RETURN on a Powerbook
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

  def mainloop
    until @done
      @terminal.draw_screen(self)

      key = @terminal.getch
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

  def run
    crash = nil

    begin
      mainloop
    rescue => e
      crash = e
    ensure
      @terminal.close

      warnings = Set.new
      warnings.merge(@terminal.warnings)
      warnings.merge(@search_editor.warnings)

      warnings.sort.each do |warning|
        $stderr.puts warning
      end

      if crash
        $stderr.puts unless warnings.empty?
        $stderr.puts("#{crash.class}: #{crash.message}")
        $stderr.puts('  ' + crash.backtrace.join("\n  "))
      end

      if crash || !warnings.empty?
        $stderr.puts
        $stderr.puts "Please report issues to #{BUGURL}"
      end

      exit 1 if crash
    end
  end
end

if __FILE__ != $PROGRAM_NAME
  # We're being required, probably due to unit testing.
  # Do nothing.
elsif $stdin.isatty
  File.open(ARGV[0], 'r') do |file|
    Moar.new(file).run
  end
else
  # Switch around some fds to enable us to read the former stdin and
  # curses to read the "real" stdin.
  stream = $stdin.clone
  $stdin.reopen(IO.new(1, 'r+'))
  Moar.new(stream).run
end
