package route

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

// Regular expression to match url parameters
var reParam *regexp.Regexp

func init() {
	reParam = regexp.MustCompile("^<([^>]+)>")
}

// Trie http://en.wikipedia.org/wiki/Trie for url matching with support of named parameters
type trie struct {
	root *trieNode
	// mapper takes the request and returns sequence that can be matched
	mapper requestMapper
}

// Takes the expression with url and the node that corresponds to this expression and returns parsed trie
func newTrieMatcher(expression string, mapper requestMapper, result *match) (*trie, error) {
	t := &trie{
		mapper: mapper,
	}
	t.root = &trieNode{trie: t}
	if len(expression) == 0 {
		return nil, fmt.Errorf("empty URL expression")
	}
	err := t.root.parseExpression(-1, expression, result)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *trie) canChain(o matcher) bool {
	_, ok := o.(*trie)
	return ok
}

func (t *trie) chain(o matcher) (matcher, error) {
	to, ok := o.(*trie)
	if !ok {
		return nil, fmt.Errorf("can chain only with other trie")
	}

	m := t.root.findMatchNode()
	m.matches = nil
	m.children = []*trieNode{to.root}
	t.root.setLevel(-1)

	return &trie{
		root:   t.root,
		mapper: newSeqMapper(t.mapper, to.mapper),
	}, nil
}

func (t *trie) String() string {
	return "trieMatcher()"
}

func (t *trie) setMatch(result *match) {
	t.root.setMatch(result)
}

// Tries can merge with other tries
func (t *trie) canMerge(m matcher) bool {
	ot, ok := m.(*trie)
	return ok && t.mapper.equivalent(ot.mapper) != nil
}

// Merge takes the other trie and modifies itself to match the passed trie as well.
// Note that trie passed as a parameter can be only simple trie without multiple branches per node, e.g. a->b->c->
// Trie on the left is "accumulating" trie that grows.
func (t *trie) merge(m matcher) (matcher, error) {
	other, ok := m.(*trie)
	if !ok {
		return nil, fmt.Errorf("can't merge %T and %T", t, m)
	}

	mapper := t.mapper.equivalent(other.mapper)
	if mapper == nil {
		return nil, fmt.Errorf("can't merge %T and %T", t, m)
	}

	root, err := t.root.merge(other.root)
	if err != nil {
		return nil, err
	}
	return &trie{root: root, mapper: mapper}, nil
}

// Takes the request and returns the location if the request path matches any of its paths
// returns nil if none of the requests matches
func (t *trie) match(r *http.Request) *match {
	if t.root == nil {
		return nil
	}

	return t.root.match(t.mapper.newIter(r))
}

type trieNode struct {
	trie *trie
	// Matching character, can be empty in case if it's a root node
	// or node with a pattern matcher
	char byte
	// Optional children of this node, can be empty if it's a leaf node
	children []*trieNode
	// If present, means that this node is a pattern matcher
	patternMatcher patternMatcher
	// If present it means this node contains potential match for a request, and this is a leaf node.
	matches []*match
	// For chained tries matching different parts of the request levels would increase for next chained trie nodes
	level int
}

func (t *trieNode) setMatch(m *match) {
	n := t.findMatchNode()
	n.matches = []*match{m}
}

func (t *trieNode) setLevel(level int) {
	if t.isRoot() {
		level++
	}
	t.level = level
	if len(t.matches) != 0 {
		return
	}

	// Check for the match in child nodes
	for _, c := range t.children {
		c.setLevel(level)
	}
}

func (t *trieNode) findMatchNode() *trieNode {
	if len(t.matches) != 0 {
		return t
	}

	// Check for the match in child nodes
	for _, c := range t.children {
		if n := c.findMatchNode(); n != nil {
			return n
		}
	}
	return nil
}

func (t *trieNode) isMatching() bool {
	return len(t.matches) != 0
}

func (t *trieNode) isRoot() bool {
	return t.char == byte(0) && t.patternMatcher == nil
}

func (t *trieNode) isPatternMatcher() bool {
	return t.patternMatcher != nil
}

//nolint:unused
func (t *trieNode) isCharMatcher() bool {
	return t.char != 0
}

func (t *trieNode) String() string {
	self := ""
	if t.patternMatcher != nil {
		self = t.patternMatcher.String()
	} else {
		self = fmt.Sprintf("%c", t.char)
	}

	if t.isMatching() {
		return fmt.Sprintf("match(%d:%s)", t.level, self)
	} else if t.isRoot() {
		return fmt.Sprintf("root(%d)", t.level)
	} else {
		return fmt.Sprintf("node(%d:%s)", t.level, self)
	}
}

func (t *trieNode) equals(o *trieNode) bool {
	return (t.level == o.level) && // we can merge nodes that are on the same level to avoid merges for different subtrie parts
		(t.char == o.char) && // chars are equal
		(t.patternMatcher == nil && o.patternMatcher == nil) || // both nodes have no matchers
		((t.patternMatcher != nil && o.patternMatcher != nil) && t.patternMatcher.equals(o.patternMatcher)) // both nodes have equal matchers
}

func (t *trieNode) merge(o *trieNode) (*trieNode, error) {
	children := make([]*trieNode, 0, len(t.children))
	merged := make(map[*trieNode]bool)

	// First, find the nodes with similar keys and merge them
	for _, c := range t.children {
		for _, c2 := range o.children {
			// The nodes are equivalent, so we can merge them
			if c.equals(c2) {
				m, err := c.merge(c2)
				if err != nil {
					return nil, err
				}
				merged[c] = true
				merged[c2] = true
				children = append(children, m)
			}
		}
	}

	// Next, append the keys that haven't been merged
	for _, c := range t.children {
		if !merged[c] {
			children = append(children, c)
		}
	}

	for _, c := range o.children {
		if !merged[c] {
			children = append(children, c)
		}
	}

	return &trieNode{
		level:          t.level,
		trie:           t.trie,
		char:           t.char,
		children:       children,
		patternMatcher: t.patternMatcher,
		matches:        append(t.matches, o.matches...),
	}, nil
}

func (t *trieNode) parseExpression(offset int, pattern string, m *match) error {
	// We are the last element, so we are the matching node
	if offset >= len(pattern)-1 {
		t.matches = []*match{m}
		return nil
	}

	// There's a next character that exists
	patternMatcher, newOffset, err := parsePatternMatcher(offset+1, pattern)
	// We have found the matcher, but the syntax or parameters are wrong
	if err != nil {
		return err
	}
	// Matcher was found
	if patternMatcher != nil {
		node := &trieNode{patternMatcher: patternMatcher, trie: t.trie}
		t.children = []*trieNode{node}
		return node.parseExpression(newOffset-1, pattern, m)
	} else {
		// Matcher was not found, next node is just a character
		node := &trieNode{char: pattern[offset+1], trie: t.trie}
		t.children = []*trieNode{node}
		return node.parseExpression(offset+1, pattern, m)
	}
}

func parsePatternMatcher(offset int, pattern string) (patternMatcher, int, error) {
	if pattern[offset] != '<' {
		return nil, -1, nil
	}

	rest := pattern[offset:]
	match := reParam.FindStringSubmatchIndex(rest)
	if len(match) == 0 {
		return nil, -1, nil
	}

	// Split parsed matcher parameters separated by :
	values := strings.Split(rest[match[2]:match[3]], ":")

	// The common syntax is <matcherType:matcherArg1:matcherArg2>
	matcherType := values[0]
	matcherArgs := values[1:]

	// In case if there's only one  <param> is implicitly converted to <string:param>
	if len(values) == 1 {
		matcherType = "string"
		matcherArgs = values
	}

	matcher, err := makeMatcher(matcherType, matcherArgs)
	if err != nil {
		return nil, offset, err
	}

	return matcher, offset + match[1], nil
}

type patternMatcher interface {
	getName() string
	match(i *charIter) bool
	equals(other patternMatcher) bool
	String() string
}

func makeMatcher(matcherType string, matcherArgs []string) (patternMatcher, error) {
	switch matcherType {
	case "string":
		return newStringMatcher(matcherArgs)
	case "path":
		return newPathMatcher(matcherArgs)
	case "int":
		return newIntMatcher(matcherArgs)
	}
	return nil, fmt.Errorf("unsupported matcher: %s", matcherType)
}

func newPathMatcher(args []string) (patternMatcher, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("expected only one parameter - variable name, got: %s", args)
	}

	return &pathMatcher{name: args[0]}, nil
}

type pathMatcher struct {
	name string
}

func (m *pathMatcher) String() string {
	return fmt.Sprintf("<path:%s>", m.name)
}

func (m *pathMatcher) getName() string {
	return m.name
}

func (m *pathMatcher) match(i *charIter) bool {
	m.grabValue(i)
	return true
}

func (m *pathMatcher) equals(other patternMatcher) bool {
	_, ok := other.(*pathMatcher)
	return ok && other.getName() == m.getName()
}

func (m *pathMatcher) grabValue(i *charIter) {
	for {
		_, _, ok := i.next()
		if !ok {
			return
		}
	}
}

func newStringMatcher(args []string) (patternMatcher, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("expected only one parameter - variable name, got: %s", args)
	}

	return &stringMatcher{name: args[0]}, nil
}

type stringMatcher struct {
	name string
}

func (s *stringMatcher) String() string {
	return fmt.Sprintf("<string:%s>", s.name)
}

func (s *stringMatcher) getName() string {
	return s.name
}

func (s *stringMatcher) match(i *charIter) bool {
	s.grabValue(i)
	return true
}

func (s *stringMatcher) equals(other patternMatcher) bool {
	_, ok := other.(*stringMatcher)
	return ok && other.getName() == s.getName()
}

func (s *stringMatcher) grabValue(i *charIter) {
	for {
		c, sep, ok := i.next()
		if !ok {
			return
		}
		if c == sep {
			i.pushBack()
			return
		}
	}
}

func newIntMatcher(args []string) (patternMatcher, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("expected only one parameter - variable name, got: %s", args)
	}

	return &intMatcher{name: args[0]}, nil
}

type intMatcher struct {
	name string
}

func (s *intMatcher) String() string {
	return fmt.Sprintf("<int:%s>", s.name)
}

func (s *intMatcher) getName() string {
	return s.name
}

func (s *intMatcher) match(iter *charIter) bool {
	// count stores amount of consumed characters,
	// so we know how many push backs to do in case there is no match
	var count int

	for {
		c, sep, ok := iter.next()
		count++

		// if the current character is not a number:
		//  - it's either a separator that means it's a match
		//  - it's some other character that means it's not a match
		if !unicode.IsDigit(rune(c)) {
			if c == sep {
				iter.pushBack()
				return true
			} else {
				for i := 0; i < count; i++ {
					iter.pushBack()
				}
				return false
			}
		}

		// if it's the end of the string, it's a match
		if !ok {
			return true
		}
	}
}

func (s *intMatcher) equals(other patternMatcher) bool {
	_, ok := other.(*intMatcher)
	return ok && other.getName() == s.getName()
}

func (t *trieNode) matchNode(i *charIter) bool {
	if i.level() != t.level {
		return false
	}

	if t.isRoot() {
		return true
	}

	if t.isPatternMatcher() {
		return t.patternMatcher.match(i)
	}

	c, _, ok := i.next()
	if !ok {
		// we have reached the end
		return false
	}

	if c != t.char {
		// no match, so don't consume the character
		i.pushBack()
		return false
	}

	return true
}

func (t *trieNode) match(i *charIter) *match {
	if !t.matchNode(i) {
		return nil
	}

	// This is a leaf node, and we are at the last character of the pattern
	if len(t.matches) != 0 && i.isEnd() {
		return t.matches[0]
	}

	// Check for the match in child nodes
	for _, c := range t.children {
		p := i.position()
		if match := c.match(i); match != nil {
			return match
		}

		i.setPosition(p)
	}
	// Child nodes did not match and we at the boundary
	if len(t.matches) != 0 && i.level() > t.level {
		return t.matches[0]
	}

	return nil
}

// printTrie is useful for debugging and test purposes,
// it outputs the formatted representation of the trie
func printTrie(t *trie) string {
	return printTrieNode(t.root)
}

func printTrieNode(e *trieNode) string {
	out := &bytes.Buffer{}
	printTrieNodeInner(out, e, 0)
	return out.String()
}

func printTrieNodeInner(b *bytes.Buffer, e *trieNode, offset int) {
	if offset == 0 {
		_, _ = fmt.Fprintf(b, "\n")
	}
	padding := strings.Repeat(" ", offset)
	_, _ = fmt.Fprintf(b, "%s%s\n", padding, e.String())
	if len(e.children) != 0 {
		for _, c := range e.children {
			printTrieNodeInner(b, c, offset+1)
		}
	}
}
