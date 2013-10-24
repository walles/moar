#!/usr/bin/ruby

require "curses"

class LineEditor
  include Curses

  attr_reader :string
  attr_reader :cursor_position

  def initialize
    @done = false
    @string = ''
    @cursor_position = 0
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
      @string << char.chr
      @cursor_position += 1
    end
    @cursor_position = [@cursor_position, 0].min
    @cursor_position = [@cursor_position, @string.length].max
  end

  def done?
    return @done
  end
end

class Terminal
  include Curses

  def initialize
    init_screen
    noecho
    stdscr.keypad(true)
    crmode
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
    status = "/#{moar.search_editor.string}"
    addstr(status)
  end

  def add_line(moar, screen_line, line)
    attrset(Curses::A_NORMAL)
    setpos(screen_line, 0)
    clrtoeol

    # Higlight search matches
    remaining = line
    printed_chars = 0
    if moar.search_editor && moar.search_editor.string.length > 0
      while true
        (head, match, tail) = remaining.partition(moar.search_editor.string)
        if match.empty?
          break
        end
        remaining = tail

        addstr(head)
        printed_chars += head.length
        attrset(A_REVERSE)
        addstr(match)
        printed_chars += match.length
        attrset(A_NORMAL)

        if printed_chars > cols
          break
        end
      end
    end

    # Print non-matching end of the line
    if printed_chars <= cols
      addstr(remaining)
      printed_chars += remaining.length
    end

    # Print a continuation character if we've printed outside the
    # window
    if printed_chars > cols
      setpos(screen_line, cols - 1)
      attrset(Curses::A_REVERSE)
      addstr(">")
    end
  end

  def add_view_status(moar)
    status = "Lines #{moar.first_line + 1}-"

    status += "#{moar.last_line + 1}"

    status += "/#{moar.lines.size}"

    percent_displayed =
      (100 * (moar.last_line + 1) / moar.lines.size).floor
    status += " #{percent_displayed}%"
    status += ", last key=#{moar.last_key}"

    attrset(Curses::A_REVERSE)
    addstr(status)
  end

  def draw_screen(moar)
    screen_line = 0

    # Draw lines
    (moar.first_line..moar.last_line).each do |line_number|
      add_line(moar, screen_line, moar.lines[line_number].strip)
      screen_line += 1
    end

    # Draw filling after EOF
    if screen_line < lines
      setpos(screen_line, 0)
      clrtoeol
      attrset(A_REVERSE)
      addstr("---")
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
    else
      abort("ERROR: Unsupported mode of operation <#{@mode}>")
    end

    refresh
  end
end

class Moar
  attr_reader :lines
  attr_reader :search_editor
  attr_reader :mode
  attr_reader :last_key

  def initialize(file)
    @terminal = Terminal.new
    @first_line = 0
    @lines = file.readlines
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

  def handle_view_keypress(key)
    case key
    when ?q.ord
      @done = true
    when ?/.ord
      @mode = :searching
      @search_editor = LineEditor.new
    when ?n.ord
      full_search
    when ?N.ord
      full_search_backwards
    when Curses::Key::RESIZE
      # Do nothing; draw_screen() will be called anyway between all
      # keypresses
    when Curses::Key::DOWN
      @first_line += 1
    when Curses::Key::NPAGE, ' '[0]
      @first_line = last_line + 1
    when Curses::Key::PPAGE
      self.last_line = first_line - 1
    when ?<.ord
      @first_line = 0
    when ?>.ord
      @first_line = @lines.size
    when Curses::Key::UP
      @first_line -= 1
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

  # Search the given line number ranges and scroll the view to show
  # the first match.
  #
  # Returns true if found and scrolled, false otherwise.
  def search_ranges(first_range, second_range)
    [first_range, second_range].each do |range|
      next unless range

      first = range.first
      last = range.last

      line_numbers = first.upto(last)
      if last < first
        line_numbers = first.downto(last)
      end

      line_numbers.each do |line_number|
        if @lines[line_number].index(@search_editor.string)
          show_line(line_number)
          return true
        end
      end
    end

    return false
  end

  # Search the full document and scroll to show the first hit
  def full_search
    return unless @search_editor
    return if @search_editor.string.empty?

    # Start searching from the first not-visible line after the
    # current screen
    first_not_visible = last_line + 1

    first_range = first_not_visible..(@lines.size - 1)
    first_range = nil unless first_not_visible <= (@lines.size - 1)

    # Wrap the search and search from the beginning until the last
    # not-visible line before the current screen
    last_not_visible = first_line - 1
    second_range = 0..last_not_visible
    second_range = nil unless last_not_visible >= 0

    search_ranges(first_range, second_range)
  end

  # Search the full document backwards and scroll to show the first
  # hit
  def full_search_backwards
    return unless @search_editor
    return if @search_editor.string.empty?

    # Start searching from the last non-visible line above the visible
    # screen
    last_not_visible = first_line - 1
    first_range = last_not_visible..0
    first_range = nil unless last_not_visible >= 0

    # Wrap the search and continue searching at the last line up to
    # the first not visible line below the current screen
    first_not_visible = last_line + 1
    second_range = (@lines.size - 1)..first_not_visible
    second_range = nil unless first_not_visible <= (@lines.size - 1)

    search_ranges(first_range, second_range)
  end

  def full_search_required?
    return false unless @search_editor
    return false if @search_editor.string.empty?

    @lines[first_line..last_line].each do |line|
      return false if line.index(@search_editor.string)
    end

    return true
  end

  def run
    begin
      while !@done
        @terminal.draw_screen(self)

        key = @terminal.getch
        case @mode
        when :viewing
          handle_view_keypress(key)
        when :searching
          @search_editor.enter_char(key)
          if full_search_required?
            full_search
          end
          if @search_editor.done?
            @mode = :viewing
          end
        else
          abort("ERROR: Unsupported mode of operation <#{@mode}>")
        end

        @last_key = key
      end
    ensure
      @terminal.close
    end
  end
end

if __FILE__ != $0
  # We're being required, probably due to unit testing.
  # Do nothing.
elsif $stdin.isatty
  File.open(ARGV[0], "r") do |file|
    Moar.new(file).run
  end
else
  # Switch around some fds to enable us to read the former stdin and
  # curses to read the "real" stdin.
  stream = $stdin.clone
  $stdin.reopen($stdout)
  Moar.new(stream).run
end
