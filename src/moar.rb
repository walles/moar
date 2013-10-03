#!/usr/bin/ruby

require "curses"

class Moar
  include Curses

  def initialize(file)
    @file = file
    @lines = IO.readlines(file)
  end

  def draw_screen()
    clear()
    setpos(0, 0)

    first_line = 0
    last_line = lines - 2
    @lines[first_line..last_line].each do |line|
      addstr(line)
    end

    addstr("Lines #{first_line + 1}-#{last_line + 1}")

    refresh()
  end

  def run
    init_screen
    noecho

    begin
      crmode
      while true
        draw_screen()

        key = getch()
        case key
        when ?q.ord
          break
        when KEY_RESIZE
          draw_screen()
        end
      end
    ensure
      close_screen
    end
  end
end

Moar.new(__FILE__).run()
