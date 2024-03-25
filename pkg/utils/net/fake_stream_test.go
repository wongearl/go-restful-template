package net

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"kubevirt.io/client-go/kubecli"
)

func TestFakeStream(t *testing.T) {
	stream := NewFakeStream(errors.New("fake"))
	assert.NotNil(t, stream.Stream(kubecli.StreamOptions{}))
	assert.Nil(t, stream.AsConn())
}
