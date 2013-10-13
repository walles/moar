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
    assert_add(test_me, Key::ENTER, 'abc', 3, true)
  end
end
