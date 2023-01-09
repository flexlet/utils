package utils

import (
	"encoding/hex"
	"fmt"
	"math/bits"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

type Bitmap struct {
	size uint32
	bits []uint64
}

func NewBitmap(size uint32) *Bitmap {
	var b Bitmap
	b.Grow(size)
	return &b
}

func loc(x uint32) (blkAt int, bitAt int) {
	blkAt = int(x >> 6)
	bitAt = int(x % 64)
	if bitAt == 0 && blkAt > 0 {
		blkAt--
	}
	if bitAt > 0 {
		bitAt--
	} else {
		bitAt = 63
	}
	return
}

func mask(bitAt int) uint64 {
	return (0x8000000000000000 >> bitAt)
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (b *Bitmap) Set(x uint32) {
	blkAt, bitAt := loc(x)
	if size := len(b.bits); blkAt >= size {
		b.grow(blkAt)
	}

	b.bits[blkAt] |= mask(bitAt)
}

// Remove removes the bit x from the bitmap, but does not shrink it.
func (b *Bitmap) Remove(x uint32) {
	blkAt, bitAt := loc(x)
	if blkAt < len(b.bits) {
		b.bits[blkAt] &^= mask(bitAt)
	}
}

// Contains checks whether a value is contained in the bitmap or not.
func (b *Bitmap) Contains(x uint32) bool {
	if x > b.size {
		return false
	}
	blkAt, bitAt := loc(x)
	return (b.bits[blkAt] & mask(bitAt)) > 0
}

// Ones sets the entire bitmap to one.
func (b *Bitmap) Ones() {
	size := len(b.bits)
	for i := 0; i < size; i++ {
		b.bits[i] = 0xffffffffffffffff
	}
}

// Min get the smallest value stored in this bitmap, assuming the bitmap is not empty.
func (b *Bitmap) Min() (uint32, bool) {
	for blkAt, blk := range b.bits {
		if blk != 0x0 {
			min := uint32(blkAt<<6 + bits.LeadingZeros64(blk) + 1)
			if min <= b.size {
				return min, true
			}
		}
	}

	return 0, false
}

// Max get the largest value stored in this bitmap, assuming the bitmap is not empty.
func (b *Bitmap) Max() (uint32, bool) {
	var blk uint64
	for blkAt := len(b.bits) - 1; blkAt >= 0; blkAt-- {
		if blk = b.bits[blkAt]; blk != 0x0 {
			max := uint32(blkAt<<6 + 64 - bits.TrailingZeros64(blk))
			if max <= b.size {
				return max, true
			}
		}
	}
	return 0, false
}

// MinZero finds the first zero bit and returns its index, assuming the bitmap is not empty.
func (b *Bitmap) MinZero() (uint32, bool) {
	for blkAt, blk := range b.bits {
		if blk != 0xffffffffffffffff {
			min := uint32(blkAt<<6 + bits.LeadingZeros64(^blk) + 1)
			if min <= b.size {
				return min, true
			}
		}
	}
	return 0, false
}

// MaxZero get the last zero bit and return its index, assuming bitmap is not empty
func (b *Bitmap) MaxZero() (uint32, bool) {
	var blk uint64
	for blkAt := len(b.bits) - 1; blkAt >= 0; blkAt-- {
		blk = ^(b.bits[blkAt])
		if blkAt == len(b.bits)-1 {
			_, bitAt := loc(b.size)
			blk >>= uint64(63 - bitAt)
			blk <<= uint64(63 - bitAt)
		}
		if blk != 0x0 {
			max := uint32(blkAt<<6 + 64 - bits.TrailingZeros64(blk))
			if max <= b.size {
				return max, true
			}
		}
	}
	return 0, false
}

// CountTo counts the number of elements in the bitmap up until the specified index. If until
// is math.MaxUint32, it will return the count. The count is non-inclusive of the index.
func (b *Bitmap) CountTo(until uint32) int {
	if len(b.bits) == 0 {
		return 0
	}

	// Figure out the index of the last block
	blkUntil, bitUntil := loc(until)
	if blkUntil >= len(b.bits) {
		blkUntil = len(b.bits) - 1
	}

	// Count the bits right before the last block
	sum := count(b.bits[:blkUntil])

	// Count the bits at the end
	sum += bits.OnesCount64(b.bits[blkUntil] >> (63 - uint64(bitUntil)))
	return sum
}

// Grow grows the bitmap size until we reach the desired bit.
func (b *Bitmap) Grow(desiredBit uint32) {
	b.size = desiredBit
	blk, _ := loc(desiredBit)
	b.grow(blk)
}

// Count returns the number of elements in this bitmap
func (b *Bitmap) Count() int {
	return b.CountTo(b.size)
}

// And computes the intersection between two bitmaps and stores the result in the current bitmap
func (a *Bitmap) And(other Bitmap, extra ...Bitmap) {
	a.size = minsize(*a, other, extra)
	max := minlen(*a, other, extra)
	a.shrink(max)
	for i := 0; i < max; i++ {
		a.bits[i] = a.bits[i] &^ other.bits[i]
	}

	for _, b := range extra {
		for i := 0; i < max; i++ {
			a.bits[i] = a.bits[i] &^ b.bits[i]
		}
	}
}

// AndNot computes the difference between two bitmaps and stores the result in the current bitmap.
// Operation works as set subtract: a - b
func (a *Bitmap) AndNot(other Bitmap, extra ...Bitmap) {
	a.size = minsize(*a, other, extra)
	max := minlen(*a, other, extra)

	for i := 0; i < max; i++ {
		a.bits[i] = a.bits[i] &^ other.bits[i]
	}

	for _, b := range extra {
		for i := 0; i < max; i++ {
			a.bits[i] = a.bits[i] &^ b.bits[i]
		}
	}

}

// Or computes the union between two bitmaps and stores the result in the current bitmap
func (a *Bitmap) Or(other Bitmap, extra ...Bitmap) {
	a.size = maxsize(*a, other, extra)
	max := maxlen(*a, other, extra)
	a.grow(max - 1)

	for i := 0; i < len(other.bits); i++ {
		a.bits[i] = a.bits[i] | other.bits[i]
	}

	for _, b := range extra {
		for i := 0; i < len(b.bits); i++ {
			a.bits[i] = a.bits[i] | b.bits[i]
		}
	}
}

// Xor computes the symmetric difference between two bitmaps and stores the result in the current bitmap
func (a *Bitmap) Xor(other Bitmap, extra ...Bitmap) {
	a.size = maxsize(*a, other, extra)
	max := maxlen(*a, other, extra)
	a.grow(max - 1)

	for i := 0; i < len(other.bits); i++ {
		a.bits[i] = a.bits[i] ^ other.bits[i]
	}

	for _, b := range extra {
		for i := 0; i < len(b.bits); i++ {
			a.bits[i] = a.bits[i] ^ b.bits[i]
		}
	}
}

// Clear clears the bitmap and resizes it to zero.
func (b *Bitmap) Clear() {
	for i := range b.bits {
		b.bits[i] = 0
	}
	b.bits = b.bits[:0]
	b.size = 0
}

// ToString returns encoded string representation for the bitmap
func (b *Bitmap) ToString() string {
	var sb strings.Builder
	for i := 0; i < len(b.bits); i++ {
		// convert each uint64 into 16 * 4-bit hexadecimal character
		writeHexdecimal(&sb, b.bits[i], true)
	}

	len := b.size >> 2
	if b.size%4 != 0 {
		len++
	}
	return sb.String()[:len]
}

// writeHexdecimal write the hexdecimal representation for given value in buffer
func writeHexdecimal(sb *strings.Builder, value uint64, pad bool) {
	maxLen := 16 // 64 bits / 4

	hexadecimal := strings.ToUpper(strconv.FormatUint(value, 16))
	hexaLen := len(hexadecimal)

	if !pad || hexaLen == maxLen {
		sb.WriteString(hexadecimal)
		return
	}

	// Add padding
	for i := hexaLen; i < maxLen; i++ {
		sb.WriteString("0")
	}

	sb.WriteString(hexadecimal)
}

// FromString decodes the received string and loads it to bitmap object
func FromString(hexString string, cap uint32) (*Bitmap, error) {
	if cap == 0 || cap > uint32(len(hexString)*4) {
		cap = uint32(len(hexString) * 4)
	}

	for len(hexString)%16 != 0 {
		hexString += "0"
	}

	bytes, err := hex.DecodeString(hexString)

	switch {
	case err != nil:
		return nil, err
	case len(bytes) == 0:
		return nil, nil
	}

	for l, r := 0, len(bytes)-1; l < r; l, r = l+1, r-1 {
		bytes[l], bytes[r] = bytes[r], bytes[l]
	}

	return FromBytes(bytes, cap), nil
}

// FromBytes reads a bitmap from a byte buffer without copying the buffer.
func FromBytes(buffer []byte, cap uint32) *Bitmap {
	if cap == 0 || cap > uint32(len(buffer)*8) {
		cap = uint32(len(buffer) * 8)
	}

	switch {
	case len(buffer) == 0:
		return nil
	case len(buffer)%8 != 0:
		panic(fmt.Sprintf("bitmap: buffer length expected to be multiple of 8, was %d", len(buffer)))
	}

	out := Bitmap{size: cap}

	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&(out.bits)))
	hdr.Len = len(buffer) >> 3
	hdr.Cap = hdr.Len
	hdr.Data = uintptr(unsafe.Pointer(&(buffer[0])))

	for l, r := 0, len(out.bits)-1; l < r; l, r = l+1, r-1 {
		out.bits[l], out.bits[r] = out.bits[r], out.bits[l]
	}

	return &out
}

// ToBytes converts the bitmap to binary representation without copying the underlying
// data. The output buffer should not be modified, since it would also change the bitmap.
func (b *Bitmap) ToBytes() (out []byte) {
	if len(b.bits) == 0 {
		return nil
	}

	bits := b.bits
	for l, r := 0, len(bits)-1; l < r; l, r = l+1, r-1 {
		bits[l], bits[r] = bits[r], bits[l]
	}

	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&out))
	hdr.Len = len(bits) * 8
	hdr.Cap = hdr.Len
	hdr.Data = uintptr(unsafe.Pointer(&(bits[0])))
	return out
}

// grow grows the size of the bitmap until we reach the desired block offset
func (b *Bitmap) grow(blkAt int) {
	if len(b.bits) > blkAt {
		return
	}

	// If there's space, resize the slice without copying.
	if cap(b.bits) > blkAt {
		b.bits = b.bits[:blkAt+1]
		return
	}

	old := b.bits
	b.bits = make([]uint64, blkAt+1, resize(cap(old), blkAt+1))
	copy(b.bits, old)
}

// shrink shrinks the size of the bitmap and resets to zero
func (b *Bitmap) shrink(length int) {
	until := len(b.bits)
	for i := length; i < until; i++ {
		b.bits[i] = 0
	}

	// Trim without reallocating
	b.bits = b.bits[:length]
}

func minsize(a, b Bitmap, extra []Bitmap) uint32 {
	size := a.size
	if size > b.size {
		size = b.size
	}
	for _, v := range extra {
		if size > v.size {
			size = b.size
		}
	}
	return size
}

// minlen calculates the minimum length of all of the bitmaps
func minlen(a, b Bitmap, extra []Bitmap) int {
	size := minint(len(a.bits), len(b.bits))
	for _, v := range extra {
		if m := minint(len(a.bits), len(v.bits)); m < size {
			size = m
		}
	}
	return size
}

func maxsize(a, b Bitmap, extra []Bitmap) uint32 {
	size := a.size
	if size < b.size {
		size = b.size
	}
	for _, v := range extra {
		if size < v.size {
			size = b.size
		}
	}
	return size
}

// maxlen calculates the maximum length of all of the bitmaps
func maxlen(a, b Bitmap, extra []Bitmap) int {
	size := maxint(len(a.bits), len(b.bits))
	for _, v := range extra {
		if m := maxint(len(a.bits), len(v.bits)); m > size {
			size = m
		}
	}
	return size
}

// maxint returns a maximum of two integers without branches.
func maxint(v1, v2 int) int {
	return v1 - ((v1 - v2) & ((v1 - v2) >> 31))
}

// minint returns a minimum of two integers without branches.
func minint(v1, v2 int) int {
	return v2 + ((v1 - v2) & ((v1 - v2) >> 31))
}

// resize calculates the new required capacity and a new index
func resize(capacity, v int) int {
	const threshold = 256
	if v < threshold {
		v |= v >> 1
		v |= v >> 2
		v |= v >> 4
		v |= v >> 8
		v |= v >> 16
		v++
		return int(v)
	}

	if capacity < threshold {
		capacity = threshold
	}

	for 0 < capacity && capacity < (v+1) {
		capacity += (capacity + 3*threshold) / 4
	}
	return capacity
}

// Count counts the number of bits set to one
func count(arr []uint64) int {
	sum := 0
	for i := 0; i < len(arr); i++ {
		sum += bits.OnesCount64(arr[i])
	}
	return sum
}
