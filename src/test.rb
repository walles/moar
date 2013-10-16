#!/usr/bin/ruby

require "#{File.dirname(__FILE__)}/moar.rb"

require "test/unit"

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
