package trie

type Node struct {
	exists bool
	value  interface{}
	child  map[rune]*Node
}

func newNode() *Node {
	return &Node{
		exists: false,
		value:  nil,
		child:  make(map[rune]*Node),
	}
}

type Trie struct {
	root *Node
	size int
}

func (t *Trie) Root() *Node {
	return t.root
}

func (t *Trie) Insert(key string) {
	curNode := t.root
	for _, v := range key {
		if curNode.child[v] == nil {
			curNode.child[v] = newNode()
		}
		curNode = curNode.child[v]
	}

	if !curNode.exists {
		t.size++
		curNode.exists = true
	}
	curNode.value = key
}

func (t *Trie) PrefixMatch(key string) []interface{} {
	node, _ := t.findNode(key)
	if node != nil {
		return t.Walk(node)
	}
	return []interface{}{}
}

func (t *Trie) Walk(node *Node) (ret []interface{}) {
	if node.exists {
		ret = append(ret, node.value)
	}
	for _, v := range node.child {
		ret = append(ret, t.Walk(v)...)
	}
	return
}

func (t *Trie) findNode(key string) (node *Node, index int) {
	curNode := t.root
	f := false
	for k, v := range key {
		if f {
			index = k
			f = false
		}
		if curNode.child[v] == nil {
			return nil, index
		}
		curNode = curNode.child[v]
		if curNode.exists {
			f = true
		}
	}

	if curNode.exists {
		index = len(key)
	}

	return curNode, index
}

func NewTrie() *Trie {
	return &Trie{
		root: newNode(),
		size: 0,
	}
}
