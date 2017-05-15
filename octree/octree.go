// Package octree provides parallelizable functions for a linear octree.
//
// Interleaved values are given as a morton encoding, also known as Z ordering.
// This is given by spacing the bits out of a uint by two bits. Naturally sorting
// the resulting uints provides locality and can be searched with common methods
// such as binary search.
package octree

// Some interesting papers:
// http://repository.upenn.edu/cgi/viewcontent.cgi?article=1167&context=meam_papers
// https://pdfs.semanticscholar.org/2243/7af0e3d86eeff22ac5d2d8d665b1561ffccf.pdf

// Dilate8 expands the bits of a uint8 with two zero bits.
func Dilate8(x uint8) uint32 {
	n := uint32(x)               // 000000000000000087654321
	n = (n ^ n<<8) & 0xf00f      // 000000008765000000004321
	n = (n ^ n<<4) & 0xc30c3     // 000087000065000043000021
	return (n ^ n<<2) & 0x249249 // 008007006005004003002001
}

// Undilate8 constricts the bits of a uint32 removing two bits starting with
// the least significant bit and every third bit there-after.
func Undilate8(x uint32) uint8 {
	n := x & 0x249249        // 008007006005004003002001
	n = (n ^ n>>2) & 0xc30c3 // 000087000065000043000021
	n = (n ^ n>>4) & 0xf00f  // 000000008765000000004321
	return uint8(n ^ n>>8)   // 000000000000000087654321
}

// Interleave8 returns x, y, z interleaved and w occupying the least significant bits.
func Interleave8(x, y, z, w uint8) uint32 {
	return uint32(w) | Dilate8(z)<<8 | Dilate8(y)<<9 | Dilate8(x)<<10
}

// Deinterleave8 deinterleaves a uint32 into four uint16s.
func Deinterleave8(a uint32) (x, y, z, w uint8) {
	return Undilate8(a >> 10), Undilate8(a >> 9), Undilate8(a >> 8), uint8(a)
}

// Dilate16 expands the bits of a uint16 with two zero bits.
func Dilate16(x uint16) uint64 {
	n := uint64(x)
	n = (n ^ n<<16) & 0xff0000ff
	n = (n ^ n<<8) & 0xf00f00f00f
	n = (n ^ n<<4) & 0xc30c30c30c3
	return (n ^ n<<2) & 0x249249249249
}

// Undilate16 constricts the bits of a uint64 removing two bits starting with
// the least significant bit and every third bit there-after.
func Undilate16(x uint64) uint16 {
	n := x & 0x249249249249
	n = (n ^ n>>2) & 0xc30c30c30c3
	n = (n ^ n>>4) & 0xf00f00f00f
	n = (n ^ n>>8) & 0xff0000ff
	return uint16(n ^ n>>16)
}

// Interleave16 returns x, y, z interleaved and w occupying the least significant bits.
func Interleave16(x, y, z, w uint16) uint64 {
	return uint64(w) | Dilate16(z)<<16 | Dilate16(y)<<17 | Dilate16(x)<<18
}

// Deinterleave16 deinterleaves a uint64 into four uint16s.
func Deinterleave16(a uint64) (x, y, z, w uint16) {
	return Undilate16(a >> 18), Undilate16(a >> 17), Undilate16(a >> 16), uint16(a)
}

// Children16 assumes n is a node in a point-based tree and subdivides n into eight equal spaces.
func Children16(n uint64) (a, b, c, d, e, f, g, h uint64) {
	lvl := uint16(n) + 1
	ds := Dilate16(1 << (16 - lvl))
	a = (n & 0xffffffffffff0000) | uint64(lvl)
	b = a + ds<<16                   // z+s
	c = a + ds<<17                   // y+s
	d = a + ds<<17 | ds<<16          // y+s, z+s
	e = a + ds<<18                   // x+s
	f = a + ds<<18 | ds<<16          // x+s, z+s
	g = a + ds<<18 | ds<<17          // x+s, y+s
	h = a + ds<<18 | ds<<17 | ds<<16 // x+s, y+s, z+s
	return
}
