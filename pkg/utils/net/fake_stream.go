package net

import (
	"net"

	"kubevirt.io/client-go/kubecli"
)

// FakeStream is a fake stream for unit test purpose
type FakeStream struct {
	err error
}

// NewFakeStream creates a instance of the FakeStream
func NewFakeStream(err error) *FakeStream {
	return &FakeStream{err: err}
}

// Stream is a fake method, do nothing but the expect error
func (f *FakeStream) Stream(options kubecli.StreamOptions) (err error) {
	err = f.err
	return
}

// AsConn is a fake method, do nothing. And return a nil.
func (f *FakeStream) AsConn() net.Conn {
	return nil
}
