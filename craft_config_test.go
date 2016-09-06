package "main"

import (
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestTester(t *testing.T) {
  assert.Equal(t, "hello", bogusTest())
}