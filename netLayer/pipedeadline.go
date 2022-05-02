package netLayer

import (
	"sync"
	"time"
)

//implements NetDeadliner. Must call InitEasyDeadline before use.
// Can be embed to a struct to make it have SetWriteDeadline, SetReadDeadline and SetDeadline method.
// And use select and ReadTimeoutChan or WriteTimeoutChan when reading or writing.
type EasyDeadline struct {
	readDeadline  PipeDeadline
	writeDeadline PipeDeadline
}

func (ed *EasyDeadline) InitEasyDeadline() {
	ed.readDeadline.cancel = make(chan struct{})
	ed.writeDeadline.cancel = make(chan struct{})
}

// try receive this to see if read timeout happens
func (ed *EasyDeadline) ReadTimeoutChan() chan struct{} {
	return ed.readDeadline.Wait()
}

// try receive this to see if write timeout happens
func (ed *EasyDeadline) WriteTimeoutChan() chan struct{} {
	return ed.writeDeadline.Wait()
}

func (ed *EasyDeadline) SetWriteDeadline(t time.Time) error {
	ed.writeDeadline.Set(t)
	return nil
}
func (ed *EasyDeadline) SetReadDeadline(t time.Time) error {
	ed.readDeadline.Set(t)
	return nil
}
func (ed *EasyDeadline) SetDeadline(t time.Time) error {
	ed.readDeadline.Set(t)
	ed.writeDeadline.Set(t)
	return nil
}

// PipeDeadline is an abstraction for handling timeouts.
//copied from golang standard package net
type PipeDeadline struct {
	mu     sync.Mutex // Guards timer and cancel
	timer  *time.Timer
	cancel chan struct{} // Must be non-nil
}

func MakePipeDeadline() PipeDeadline {
	return PipeDeadline{cancel: make(chan struct{})}
}

// set sets the point in time when the deadline will time out.
// A timeout event is signaled by closing the channel returned by waiter.
// Once a timeout has occurred, the deadline can be refreshed by specifying a
// t value in the future.
//
// A zero value for t prevents timeout.
func (d *PipeDeadline) Set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil && !d.timer.Stop() {
		<-d.cancel // Wait for the timer callback to finish and close cancel
	}
	d.timer = nil

	// Time is zero, then there is no deadline.
	closed := isClosedChan(d.cancel)
	if t.IsZero() {
		if closed {
			d.cancel = make(chan struct{})
		}
		return
	}

	// Time in the future, setup a timer to cancel in the future.
	if dur := time.Until(t); dur > 0 {
		if closed {
			d.cancel = make(chan struct{})
		}
		d.timer = time.AfterFunc(dur, func() {
			close(d.cancel)
		})
		return
	}

	// Time in the past, so close immediately.
	if !closed {
		close(d.cancel)
	}
}

// wait returns a channel that is closed when the deadline is exceeded.
func (d *PipeDeadline) Wait() chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cancel
}

func isClosedChan(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
