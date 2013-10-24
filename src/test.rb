#!/usr/bin/ruby

require 'pathname'
require "test/unit"

require "#{Pathname(__FILE__).realpath.dirname}/moar.rb"

class TestLineEditor < Test::Unit::TestCase
  include Curses

  def assert_add(test_me, key, string, cursor_pos, done)
    test_me.enter_char(key)
    assert_equal(string, test_me.string, "string")
    assert_equal(cursor_pos, test_me.cursor_position, "cursor position")
    assert_equal(done, test_me.done?, "done")
  end

  def test_basic()
    test_me = LineEditor.new()
    assert_equal('', test_me.string)
    assert_equal(0, test_me.cursor_position)
    assert(!test_me.done?)

    assert_add(test_me, ?a.ord, 'a', 1, false)
    assert_add(test_me, ?b.ord, 'ab', 2, false)
    assert_add(test_me, ?c.ord, 'abc', 3, false)

    # 127 = BACKSPACE on a Powerbook.  Key::BACKSPACE is something
    # else, don't know why they aren't one and the same.
    assert_add(test_me, 127, 'ab', 2, false)

    # 10 == RETURN on a Powerbook.  Key::ENTER is something else,
    # don't know why they aren't one and the same.
    assert_add(test_me, 10, 'ab', 2, true)
  end

  # Verify that we become done after backspacing out of an empty
  # line
  def test_done_on_empty_backspace()
    test_me = LineEditor.new()
    assert_add(test_me, 127, '', 0, true)
  end
end

class MockTerminal
  # We can display two lines
  def lines; 2 end
end

class TestMoar < Test::Unit::TestCase
  def test_line_methods
    # This method assumes the MockTerminal can display two lines
    terminal = MockTerminal.new
    test_me = Moar.new(['1', '2', '3', '4'], terminal)

    assert_equal(0, test_me.first_line)
    assert_equal(1, test_me.last_line)

    assert_equal(2, test_me.last_line(1))

    test_me.last_line = 2
    assert_equal(1, test_me.first_line)
    assert_equal(2, test_me.last_line)
  end
end
