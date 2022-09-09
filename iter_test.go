package route

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type IterSuite struct {
	suite.Suite
}

func TestIterSuite(t *testing.T) {
	suite.Run(t, new(IterSuite))
}

func (s *IterSuite) TestEmptyOperationsSucceed() {
	var values []string
	var seps []byte
	i := newIter(values, seps)
	_, _, ok := i.next()
	s.Equal(false, ok)

	_, _, ok = i.next()
	s.Equal(false, ok)
}

func (s *IterSuite) TestUnwind() {
	tc := []charTc{
		{
			name:  "Simple iteration",
			input: []string{"hello"},
			sep:   []byte{pathSep},
		},
		{
			name:  "Combined iteration",
			input: []string{"hello", "world", "ha"},
			sep:   []byte{pathSep, domainSep, domainSep},
		},
	}
	for _, test := range tc {
		i := newIter(test.input, test.sep)
		var out []byte
		for {
			ch, _, ok := i.next()
			if !ok {
				break
			}
			out = append(out, ch)
		}

		s.Equal(test.String(), string(out), "%v", test.name)
	}
}

func (s *IterSuite) TestRecoverPosition() {
	i := newIter([]string{"hi", "world"}, []byte{pathSep, domainSep})
	i.next()
	i.next()
	p := i.position()
	i.next()
	i.setPosition(p)

	ch, sep, ok := i.next()
	s.True(ok)
	s.Equal(byte('w'), ch)
	s.Equal(byte(domainSep), sep)
}

func (s *IterSuite) TestPushBack() {
	i := newIter([]string{"hi", "world"}, []byte{pathSep, domainSep})
	i.pushBack()
	i.pushBack()
	ch, sep, ok := i.next()
	s.True(ok)
	s.Equal(byte('h'), ch)
	s.Equal(byte(pathSep), sep)
}

func (s *IterSuite) TestPushBackBoundary() {
	i := newIter([]string{"hi", "world"}, []byte{pathSep, domainSep})
	i.next()
	i.next()
	i.next()
	i.pushBack()
	i.pushBack()
	ch, sep, ok := i.next()
	s.True(ok)
	s.Equal("i", fmt.Sprintf("%c", ch))
	s.Equal(fmt.Sprintf("%c", pathSep), fmt.Sprintf("%c", sep))
}

func (s *IterSuite) TestString() {
	i := newIter([]string{"hi"}, []byte{pathSep})
	i.next()
	s.Equal("<1:hi>", i.String())
	i.next()
	s.Equal("<end>", i.String())
}

type charTc struct {
	name  string
	input []string
	sep   []byte
}

func (c *charTc) String() string {
	return strings.Join(c.input, "")
}
