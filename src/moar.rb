#!/usr/bin/ruby

require "curses"

class Moar
  include Curses

  def initialize(file)
    @first_line = 0
    @lines = file.readlines
    @last_key = 0
  end

  def draw_screen()
    # @first_line must not be closer than lines-2 from the end
    max_first_line = @lines.size - (lines - 1)
    @first_line = [@first_line, max_first_line].min()

    # @first_line cannot be negative
    @first_line = [0, @first_line].max()

    clear()
    setpos(0, 0)

    attrset(A_NORMAL)
    last_line = @first_line + lines - 2
    for line_number in @first_line..last_line do
      if line_number < @lines.size
        addstr(@lines[line_number])
      else
        addstr("~\n")
      end
    end

    attrset(A_REVERSE)

    status = "Lines #{@first_line + 1}-"

    last_displayed_line = [@lines.size, last_line + 1].min()
    status += "#{last_displayed_line}"

    status += "/#{@lines.size}"

    percent_displayed =
      ((100 * last_displayed_line) / @lines.size()).floor()
    status += " #{percent_displayed}%"
    status += ", last key=#{@last_key}"
    addstr(status)

    refresh()
  end

  def run
    init_screen
    noecho
    stdscr.keypad(true)

    begin
      crmode
      while true
        draw_screen()

        key = getch()
        case key
        when ?q.ord
          break
        when Key::RESIZE
          draw_screen()
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

        @last_key = key
      end
    ensure
      close_screen
    end
  end
end

if $stdin.isatty()
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
