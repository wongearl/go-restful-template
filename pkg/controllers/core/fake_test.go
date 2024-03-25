package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNoErrors(t *testing.T) {
	assert.True(t, NoErrors(t, nil))
}
