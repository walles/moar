#!/usr/bin/ruby

require "curses"

class LineEditor
  include Curses

  attr_reader :string
  attr_reader :cursor_position

  def initialize()
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
    else
      @string << char.chr
      @cursor_position += 1
    end
  end

  def done?()
    return @done
  end
end

class Moar
  include Curses

  def initialize(file)
    @first_line = 0
    @lines = file.readlines
    @last_key = 0
    @done = false

    # Mode can be :viewing and :searching
    @mode = :viewing
  end

  def add_view_status()
    status = "Lines #{@first_line + 1}-"

    last_displayed_line = [@lines.size, @last_line + 1].min()
    status += "#{last_displayed_line}"

    status += "/#{@lines.size}"

    percent_displayed =
      ((100 * last_displayed_line) / @lines.size()).floor()
    status += " #{percent_displayed}%"
    status += ", last key=#{@last_key}"

    attrset(A_REVERSE)
    addstr(status)
  end

  def add_search_status()
    status = "/#{@line_editor.string}"
    addstr(status)
  end

  def add_line(screen_line, line)
    attrset(A_NORMAL)
    setpos(screen_line, 0)

    if line.length <= cols
      addstr(line)
    else
      line = line[0..(cols - 2)]
      addstr(line)

      attrset(A_REVERSE)
      addstr(">")
    end
  end

  def draw_screen()
    # @first_line must not be closer than lines-2 from the end
    max_first_line = @lines.size - (lines - 1)
    @first_line = [@first_line, max_first_line].min()

    # @first_line cannot be negative
    @first_line = [0, @first_line].max()

    clear()

    screen_line = 0
    @last_line = @first_line + lines - 2
    for line_number in @first_line..@last_line do
      if line_number < @lines.size
        add_line(screen_line, @lines[line_number].strip)
      else
        addstr("~\n")
      end
      screen_line += 1
    end

    setpos(lines - 1, 0)
    case @mode
    when :viewing
      add_view_status()
    when :searching
      add_search_status()
    else
      abort("ERROR: Unsupported mode of operation <#{@mode}>")
    end

    refresh()
  end

  def handle_view_keypress(key)
    case key
    when ?q.ord
      @done = true
    when ?/.ord
      @mode = :searching
      @line_editor = LineEditor.new()
    when Key::RESIZE
      # Do nothing; draw_screen() will be called anyway between all
      # keypresses
    when Key::DOWN
      @first_line += 1
    when Key::NPAGE, ' '[0]
      @first_line += lines - 1
    when Key::PPAGE
      @first_line -= lines - 1
    when ?<.ord
      @first_line = 0
    when ?>.ord
      @first_line = @lines.size()
    when Key::UP
      @first_line -= 1
    end
  end

  def run
    init_screen
    noecho
    stdscr.keypad(true)

    begin
      crmode
      while !@done
        draw_screen()

        key = getch()
        case @mode
        when :viewing
          handle_view_keypress(key)
        when :searching
          @line_editor.enter_char(key)
          if @line_editor.done?
            @mode = :viewing
            @line_editor = nil
          end
        else
          abort("ERROR: Unsupported mode of operation <#{@mode}>")
        end

        @last_key = key
      end
    ensure
      close_screen
    end
  end
end

if __FILE__ != $0
  # We're being required, probably due to unit testing.
  # Do nothing.
elsif $stdin.isatty()
  File.open(ARGV[0], "r") do |file|
    Moar.new(file).run()
  end
else
  # Switch around some fds to enable us to read the former stdin and
  # curses to read the "real" stdin.
  stream = $stdin.clone()
  $stdin.reopen($stdout)
  Moar.new(stream).run()
end
