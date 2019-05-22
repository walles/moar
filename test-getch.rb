#!/usr/bin/env ruby
# frozen_string_literal: true

# Script for interactively testing wide_getch() output in moar.rb

require 'io/console'
require 'io/wait'

def wide_getch()
  char = STDIN.getch

  return char unless char == "\e"

  return char unless STDIN.ready?

  return char + STDIN.getch + STDIN.getch
end

response = wide_getch
puts("Length: #{response.length}  Bytesize: #{response.bytesize}")
response.each_char { |x| puts "#{x} = #{x.ord}  #{x.encoding}" }
