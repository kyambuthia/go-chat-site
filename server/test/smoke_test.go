package test

import "testing"

func TestSanity(t *testing.T) {
  if 1+1 != 2 {
    t.Fatal("math broken")
  }
}
