package cidranger

import (
	"fmt"
	"net"
	"strings"

	rnet "github.com/yl2chen/cidranger/net"
)

// prefixTrie is a path-compressed (PC) trie implementation of the
// ranger interface inspired by this blog post:
// https://vincent.bernat.im/en/blog/2017-ipv4-route-lookup-linux
//
// CIDR blocks are stored using a prefix tree structure where each node has its
// parent as prefix, and the path from the root node represents current CIDR
// block.
//
// For IPv4, the trie structure guarantees max depth of 32 as IPv4 addresses are
// 32 bits long and each bit represents a prefix tree starting at that bit. This
// property also guarantees constant lookup time in Big-O notation.
//
// Path compression compresses a string of node with only 1 child into a single
// node, decrease the amount of lookups necessary during containment tests.
//
// Level compression dictates the amount of direct children of a node by
// allowing it to handle multiple bits in the path.  The heuristic (based on
// children population) to decide when the compression and decompression happens
// is outlined in the prior linked blog, and will be experimented with in more
// depth in this project in the future.
//
// Note: Can not insert both IPv4 and IPv6 network addresses into the same
// prefix trie, use versionedRanger wrapper instead.
//
// TODO: Implement level-compressed component of the LPC trie.
type prefixTrie struct {
	parent   *prefixTrie
	children []*prefixTrie

	numBitsSkipped uint
	numBitsHandled uint

	network rnet.Network
	entry   RangerEntry

	size int // This is only maintained in the root trie.
}

// newPrefixTree creates a new prefixTrie.
func newPrefixTree(version rnet.IPVersion) Ranger {
	_, rootNet, _ := net.ParseCIDR("0.0.0.0/0")
	if version == rnet.IPv6 {
		_, rootNet, _ = net.ParseCIDR("0::0/0")
	}
	return &prefixTrie{
		children:       make([]*prefixTrie, 2, 2),
		numBitsSkipped: 0,
		numBitsHandled: 1,
		network:        rnet.NewNetwork(*rootNet),
	}
}

func newPathprefixTrie(network rnet.Network, numBitsSkipped uint) *prefixTrie {
	version := rnet.IPv4
	if len(network.Number) == rnet.IPv6Uint32Count {
		version = rnet.IPv6
	}
	path := newPrefixTree(version).(*prefixTrie)
	path.numBitsSkipped = numBitsSkipped
	path.network = network.Masked(int(numBitsSkipped))
	return path
}

func newEntryTrie(network rnet.Network, entry RangerEntry) *prefixTrie {
	ones, _ := network.IPNet.Mask.Size()
	leaf := newPathprefixTrie(network, uint(ones))
	leaf.entry = entry
	return leaf
}

// Insert inserts a RangerEntry into prefix trie.
func (p *prefixTrie) Insert(entry RangerEntry) error {
	network := entry.Network()
	sizeIncreased, err := p.insert(rnet.NewNetwork(network), entry)
	if sizeIncreased {
		p.size++
	}
	return err
}

// Remove removes RangerEntry identified by given network from trie.
func (p *prefixTrie) Remove(network net.IPNet) (RangerEntry, error) {
	entry, err := p.remove(rnet.NewNetwork(network))
	if entry != nil {
		p.size--
	}
	return entry, err
}

// Contains returns boolean indicating whether given ip is contained in any
// of the inserted networks.
func (p *prefixTrie) Contains(ip net.IP) (bool, error) {
	nn := rnet.NewNetworkNumber(ip)
	if nn == nil {
		return false, ErrInvalidNetworkNumberInput
	}
	return p.contains(nn)
}

// ContainingNetworks returns the list of RangerEntry(s) the given ip is
// contained in in ascending prefix order.
func (p *prefixTrie) ContainingNetworks(ip net.IP) ([]RangerEntry, error) {
	nn := rnet.NewNetworkNumber(ip)
	if nn == nil {
		return nil, ErrInvalidNetworkNumberInput
	}
	return p.containingNetworks(nn)
}

// CoveredNetworks returns the list of RangerEntry(s) the given ipnet
// covers.  That is, the networks that are completely subsumed by the
// specified network.
func (p *prefixTrie) CoveredNetworks(network net.IPNet) ([]RangerEntry, error) {
	net := rnet.NewNetwork(network)
	return p.coveredNetworks(net)
}

// Len returns number of networks in ranger.
func (p *prefixTrie) Len() int {
	return p.size
}

// String returns string representation of trie, mainly for visualization and
// debugging.
func (p *prefixTrie) String() string {
	children := []string{}
	padding := strings.Repeat("| ", p.level()+1)
	for bits, child := range p.children {
		if child == nil {
			continue
		}
		childStr := fmt.Sprintf("\n%s%d--> %s", padding, bits, child.String())
		children = append(children, childStr)
	}
	return fmt.Sprintf("%s (target_pos:%d:has_entry:%t)%s", p.network,
		p.targetBitPosition(), p.hasEntry(), strings.Join(children, ""))
}

func (p *prefixTrie) contains(number rnet.NetworkNumber) (bool, error) {
	if !p.network.Contains(number) {
		return false, nil
	}
	if p.hasEntry() {
		return true, nil
	}
	if p.targetBitPosition() < 0 {
		return false, nil
	}
	bit, err := p.targetBitFromIP(number)
	if err != nil {
		return false, err
	}
	child := p.children[bit]
	if child != nil {
		return child.contains(number)
	}
	return false, nil
}

func (p *prefixTrie) containingNetworks(number rnet.NetworkNumber) ([]RangerEntry, error) {
	results := []RangerEntry{}
	if !p.network.Contains(number) {
		return results, nil
	}
	if p.hasEntry() {
		results = []RangerEntry{p.entry}
	}
	if p.targetBitPosition() < 0 {
		return results, nil
	}
	bit, err := p.targetBitFromIP(number)
	if err != nil {
		return nil, err
	}
	child := p.children[bit]
	if child != nil {
		ranges, err := child.containingNetworks(number)
		if err != nil {
			return nil, err
		}
		if len(ranges) > 0 {
			if len(results) > 0 {
				results = append(results, ranges...)
			} else {
				results = ranges
			}
		}
	}
	return results, nil
}

func (p *prefixTrie) coveredNetworks(network rnet.Network) ([]RangerEntry, error) {
	var results []RangerEntry
	if network.Covers(p.network) {
		for entry := range p.walkDepth() {
			results = append(results, entry)
		}
	} else if p.targetBitPosition() >= 0 {
		bit, err := p.targetBitFromIP(network.Number)
		if err != nil {
			return results, err
		}
		child := p.children[bit]
		if child != nil {
			return child.coveredNetworks(network)
		}
	}
	return results, nil
}

func (p *prefixTrie) insert(network rnet.Network, entry RangerEntry) (bool, error) {
	if p.network.Equal(network) {
		sizeIncreased := p.entry == nil
		p.entry = entry
		return sizeIncreased, nil
	}

	bit, err := p.targetBitFromIP(network.Number)
	if err != nil {
		return false, err
	}
	existingChild := p.children[bit]

	// No existing child, insert new leaf trie.
	if existingChild == nil {
		p.appendTrie(bit, newEntryTrie(network, entry))
		return true, nil
	}

	// Check whether it is necessary to insert additional path prefix between current trie and existing child,
	// in the case that inserted network diverges on its path to existing child.
	lcb, err := network.LeastCommonBitPosition(existingChild.network)
	divergingBitPos := int(lcb) - 1
	if divergingBitPos > existingChild.targetBitPosition() {
		pathPrefix := newPathprefixTrie(network, p.totalNumberOfBits()-lcb)
		err := p.insertPrefix(bit, pathPrefix, existingChild)
		if err != nil {
			return false, err
		}
		// Update new child
		existingChild = pathPrefix
	}
	return existingChild.insert(network, entry)
}

func (p *prefixTrie) appendTrie(bit uint32, prefix *prefixTrie) {
	p.children[bit] = prefix
	prefix.parent = p
}

func (p *prefixTrie) insertPrefix(bit uint32, pathPrefix, child *prefixTrie) error {
	// Set parent/child relationship between current trie and inserted pathPrefix
	p.children[bit] = pathPrefix
	pathPrefix.parent = p

	// Set parent/child relationship between inserted pathPrefix and original child
	pathPrefixBit, err := pathPrefix.targetBitFromIP(child.network.Number)
	if err != nil {
		return err
	}
	pathPrefix.children[pathPrefixBit] = child
	child.parent = pathPrefix
	return nil
}

func (p *prefixTrie) remove(network rnet.Network) (RangerEntry, error) {
	if p.hasEntry() && p.network.Equal(network) {
		entry := p.entry
		p.entry = nil

		err := p.compressPathIfPossible()
		if err != nil {
			return nil, err
		}
		return entry, nil
	}
	if p.targetBitPosition() < 0 {
		return nil, nil
	}
	bit, err := p.targetBitFromIP(network.Number)
	if err != nil {
		return nil, err
	}
	child := p.children[bit]
	if child != nil {
		return child.remove(network)
	}
	return nil, nil
}

func (p *prefixTrie) qualifiesForPathCompression() bool {
	// Current prefix trie can be path compressed if it meets all following.
	//		1. records no CIDR entry
	//		2. has single or no child
	//		3. is not root trie
	return !p.hasEntry() && p.childrenCount() <= 1 && p.parent != nil
}

func (p *prefixTrie) compressPathIfPossible() error {
	if !p.qualifiesForPathCompression() {
		// Does not qualify to be compressed
		return nil
	}

	// Find lone child.
	var loneChild *prefixTrie
	for _, child := range p.children {
		if child != nil {
			loneChild = child
			break
		}
	}

	// Find root of currnt single child lineage.
	parent := p.parent
	for ; parent.qualifiesForPathCompression(); parent = parent.parent {
	}
	parentBit, err := parent.targetBitFromIP(p.network.Number)
	if err != nil {
		return err
	}
	parent.children[parentBit] = loneChild

	// Attempts to furthur apply path compression at current lineage parent, in case current lineage
	// compressed into parent.
	return parent.compressPathIfPossible()
}

func (p *prefixTrie) childrenCount() int {
	count := 0
	for _, child := range p.children {
		if child != nil {
			count++
		}
	}
	return count
}

func (p *prefixTrie) totalNumberOfBits() uint {
	return rnet.BitsPerUint32 * uint(len(p.network.Number))
}

func (p *prefixTrie) targetBitPosition() int {
	return int(p.totalNumberOfBits()-p.numBitsSkipped) - 1
}

func (p *prefixTrie) targetBitFromIP(n rnet.NetworkNumber) (uint32, error) {
	// This is a safe uint boxing of int since we should never attempt to get
	// target bit at a negative position.
	return n.Bit(uint(p.targetBitPosition()))
}

func (p *prefixTrie) hasEntry() bool {
	return p.entry != nil
}

func (p *prefixTrie) level() int {
	if p.parent == nil {
		return 0
	}
	return p.parent.level() + 1
}

// walkDepth walks the trie in depth order, for unit testing.
func (p *prefixTrie) walkDepth() <-chan RangerEntry {
	entries := make(chan RangerEntry)
	go func() {
		if p.hasEntry() {
			entries <- p.entry
		}
		childEntriesList := []<-chan RangerEntry{}
		for _, trie := range p.children {
			if trie == nil {
				continue
			}
			childEntriesList = append(childEntriesList, trie.walkDepth())
		}
		for _, childEntries := range childEntriesList {
			for entry := range childEntries {
				entries <- entry
			}
		}
		close(entries)
	}()
	return entries
}
