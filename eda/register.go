// Copyright 2020 The go-lpc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package eda

import (
	"io"
	"unsafe"

	"github.com/go-lpc/mim/eda/internal/regs"
)

func unsafeAdd(ptr unsafe.Pointer, n int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(ptr) + uintptr(n))
}

func unsafeSlice(ptr *byte, n int) []byte {
	return (*[4 + nHR*nBytesCfgHR]byte)(unsafe.Pointer(ptr))[:n]
}

type rwer interface {
	io.ReaderAt
	io.WriterAt
	Bytes() []byte
}

type reg32 struct {
	ptr *uint32
	r   func() uint32
	w   func(v uint32)
}

func newReg32(rw rwer, offset int64) reg32 {
	raw := rw.Bytes()
	ptr := unsafe.Pointer(&raw[offset])
	reg := reg32{ptr: (*uint32)(ptr)}
	reg.r = reg.rImpl
	reg.w = reg.wImpl
	return reg
}

func (x *reg32) rImpl() uint32  { return *x.ptr }
func (x *reg32) wImpl(v uint32) { *x.ptr = v }

type hrCfg struct {
	ptr unsafe.Pointer
}

func newHRCfg(rw rwer, offset int64) hrCfg {
	raw := rw.Bytes()
	ptr := unsafe.Pointer(&raw[offset])
	return hrCfg{ptr}
}

func (hr *hrCfg) r(i int) byte {
	ptr := unsafeAdd(hr.ptr, i)
	return *(*byte)(ptr)
}

func (hr *hrCfg) w(p []byte) (int, error) {
	n := copy(unsafeSlice((*byte)(hr.ptr), len(p)), p)
	return n, nil
}

type daqFIFO struct {
	pins [6]reg32
}

func newDAQFIFO(rw rwer, offset int64) daqFIFO {
	const sz = int64(unsafe.Sizeof(uint32(0)))
	return daqFIFO{
		pins: [6]reg32{
			newReg32(rw, offset+sz*regs.ALTERA_AVALON_FIFO_LEVEL_REG),
			newReg32(rw, offset+sz*regs.ALTERA_AVALON_FIFO_STATUS_REG),
			newReg32(rw, offset+sz*regs.ALTERA_AVALON_FIFO_EVENT_REG),
			newReg32(rw, offset+sz*regs.ALTERA_AVALON_FIFO_IENABLE_REG),
			newReg32(rw, offset+sz*regs.ALTERA_AVALON_FIFO_ALMOSTFULL_REG),
			newReg32(rw, offset+sz*regs.ALTERA_AVALON_FIFO_ALMOSTEMPTY_REG),
		},
	}
}

func (daq *daqFIFO) r(i int) uint32 {
	return daq.pins[i].r()
}

func (daq *daqFIFO) w(i int, v uint32) {
	daq.pins[i].w(v)
}
