package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMessage(t *testing.T) {
	message := GetMessage()
	assert.Equal(t, "Hello World", message)
}
