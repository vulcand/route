package route

import (
	"fmt"
)

// charPos stores the position in the iterator
type charPos struct {
	i  int
	si int
}

// charIter is an iterator over sequence of strings, returns byte-by-byte characters in string by string
type charIter struct {
	i  int // position in the current string
	si int // position in the array of strings

	seq []string // sequence of strings, e.g. ["GET", "/path"]
	sep []byte   // every string in the sequence has an associated separator used for trie matching, e.g. path uses '/' for separator
	// so sequence ["a.host", "/path "]has accompanying separators ['.', '/']
}

func newIter(seq []string, sep []byte) *charIter {
	return &charIter{
		i:   0,
		si:  0,
		seq: seq,
		sep: sep,
	}
}

func (c *charIter) level() int {
	return c.si
}

func (c *charIter) String() string {
	if c.isEnd() {
		return "<end>"
	}
	return fmt.Sprintf("<%d:%v>", c.i, c.seq[c.si])
}

func (c *charIter) isEnd() bool {
	return len(c.seq) == 0 || // no data at all
		(c.si >= len(c.seq)-1 && c.i >= len(c.seq[c.si])) || // we are at the last char of last seq
		(len(c.seq[c.si]) == 0) // empty input
}

func (c *charIter) position() charPos {
	return charPos{i: c.i, si: c.si}
}

func (c *charIter) setPosition(p charPos) {
	c.i = p.i
	c.si = p.si
}

func (c *charIter) pushBack() {
	if c.i == 0 && c.si == 0 { // this is start
		return
	} else if c.i == 0 && c.si != 0 { // this is start of the next string
		c.si--
		c.i = len(c.seq[c.si]) - 1
		return
	}
	c.i--
}

// next returns current byte in the sequence, separator corresponding to that byte, and boolean indicator of whether it's the end of the sequence
func (c *charIter) next() (byte, byte, bool) {
	// we have reached the last string in the index, end
	if c.isEnd() {
		return 0, 0, false
	}

	b := c.seq[c.si][c.i]
	sep := c.sep[c.si]
	c.i++

	// current string index exceeded the last char of the current string
	// move to the next string if it's present
	if c.i >= len(c.seq[c.si]) && c.si < len(c.seq)-1 {
		c.si++
		c.i = 0
	}

	return b, sep, true
}
