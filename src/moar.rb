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

        @first_line = 0 if @first_line < 0

        @last_key = key
      end
    ensure
      close_screen
    end
  end
end

Moar.new(__FILE__).run()
