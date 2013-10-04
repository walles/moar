#!/usr/bin/ruby

require "curses"

class Moar
  include Curses

  def initialize(file)
    @first_line = 0
    @file = file
    @lines = IO.readlines(file)
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
    @lines[@first_line..last_line].each do |line|
      addstr(line)
    end

    attrset(A_REVERSE)
    addstr("Lines #{@first_line + 1}-#{last_line + 1}, key=#{@last_key}")

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

Moar.new(__FILE__).run()
