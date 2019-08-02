package utils

import "testing"

func TestIncludes(t *testing.T) {
	if !Includes([]string{"foo", "bar"}, "foo") {
		t.Errorf("foo is included")
	}
	if !Includes([]string{"foo", "bar"}, "bar") {
		t.Errorf("bar is included")
	}
	if Includes([]string{"foo", "bar"}, "baz") {
		t.Errorf("baz is not included")
	}
}
