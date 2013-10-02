#!/usr/bin/ruby

require "curses"

class Moar
  include Curses

  def initialize(file)
    @file = file
    @lines = IO.readlines(file)
  end

  def draw_screen()
    setpos(0, 0)
    @lines.each do |line|
      addstr(line)
    end

    setpos((lines - 5) / 2, (cols - 10) / 2)
    addstr("Hit any key, or q to quit")
  end

  def run
    init_screen
    noecho

    begin
      crmode
      draw_screen()

      refresh
      while true
        key = getch()
        break if key == ?q.ord
        addstr("key=#{key}\n")
      end
    ensure
      close_screen
    end
  end
end

Moar.new(__FILE__).run()
