package main

import (
	"bufio"
	"encoding/json"
	"io"
	"runtime"
	"strings"

	"github.com/gonum/graph/path"
	"github.com/gonum/graph/simple"
)

type IdiomNode struct {
	id    int
	Idiom string
}

func (this *IdiomNode) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.Idiom)
}

func (this *IdiomNode) ID() int {
	return this.id
}

type beIndexKey struct {
	b rune
	e rune
}

/* IdiomGraph 成语组成的图, 当一个 IdiomGraph 对象被创建初始化之后便只允许只读操作, 因此一个
IdiomGraph 对象可以被多个 goroutine 同时访问.
*/
type IdiomGraph struct {
	idioms map[string]*IdiomNode

	bindex  map[rune][]*IdiomNode
	eindex  map[rune][]*IdiomNode
	beindex map[beIndexKey][]*IdiomNode

	// TODO(pp-qq): 忽然意识到 IdiomGraph 本身就可以作为一个图结构了, 可以直接实现 graph.Graph 的接口了
	// 而没必要再使用 simple.DirectedGraph 来存储了. 周末可以改一下.
	graph          *simple.DirectedGraph
	shortest_paths path.AllShortest
}

func IsValidIdiom(str string) bool {
	// 之后可以利用 unicode.Is(unicode.Scripts["Han"], r) 来判断 str 中是否全是汉字等.
	words := []rune(str)
	return len(words) >= 2
}

const (
	// graph 中边的 weight 的取值定义.
	// kSelfWeight, kAbsentWeight 对应着 simple.NewDirectedGraph() 中的相应参数.
	// kNormalWeight 中 simple.Edge.W 的取值.
	kSelfWeight   = 12138.0
	kNormalWeight = 23333.0
	kAbsentWeight = 66666.0
)

/* 从指定的 reader 中加载成语并根据加载的成语创建一个 IdiomGraph 对象.

reader 的格式应该是一行一个成语, 不合法的成语将被忽略.
*/
func LoadIdiomGraph(input io.Reader, gcnum int) (*IdiomGraph, error) {
	// graph_id 存放着下一个 node 的 id.
	graph_id := 0

	// 以下成语数目, 汉字数目参考: http://baike.baidu.com/item/%E6%B1%89%E8%AF%AD%E5%A4%A7%E8%BE%9E%E5%85%B8
	this := &IdiomGraph{
		idioms:  make(map[string]*IdiomNode, 55555),
		bindex:  make(map[rune][]*IdiomNode, 23333),
		eindex:  make(map[rune][]*IdiomNode, 23333),
		beindex: make(map[beIndexKey][]*IdiomNode, 55555),
	}

	bufreader := bufio.NewReader(input)
	for {
		idiom, err := bufreader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		idiom = strings.TrimSpace(idiom)
		if IsValidIdiom(idiom) {
			this.idioms[idiom] = &IdiomNode{id: graph_id, Idiom: idiom}
			graph_id++
		}
		if err != nil {
			break
		}
	}
	runtime.GC()

	this.graph = simple.NewDirectedGraph(kSelfWeight, kAbsentWeight)
	gc_counter := 0
	for idiom, idiom_node := range this.idioms {
		idiom_words := []rune(idiom)
		bword, eword := idiom_words[0], idiom_words[len(idiom_words)-1]

		this.graph.AddNode(idiom_node)
		for _, from_node := range this.eindex[bword] {
			this.graph.SetEdge(&simple.Edge{from_node, idiom_node, kNormalWeight})
		}
		for _, to_node := range this.bindex[eword] {
			this.graph.SetEdge(&simple.Edge{idiom_node, to_node, kNormalWeight})
		}

		beindex_key := beIndexKey{bword, eword}
		this.bindex[bword] = append(this.bindex[bword], idiom_node)
		this.eindex[eword] = append(this.eindex[eword], idiom_node)
		this.beindex[beindex_key] = append(this.beindex[beindex_key], idiom_node)

		gc_counter++
		if gc_counter >= gcnum {
			gc_counter = 0
			runtime.GC()
		}
	}
	runtime.GC()

	this.shortest_paths = path.DijkstraAllPaths(this.graph)
	runtime.GC()
	return this, nil
}

// 返回 idiom 在 graph 中对应的 node, 若不存在则返回 nil.
func (this *IdiomGraph) Find(idiom string) *IdiomNode {
	return this.idioms[idiom]
}

func normalize(offset, length, slicelen int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if length < 0 {
		length = slicelen
	}
	offsetend := offset + length
	if offset > slicelen {
		offset = slicelen
	}
	if offsetend > slicelen {
		offsetend = slicelen
	}
	return offset, offsetend
}

/* 获取以 word 开头的成语列表.

word; 存放着一个汉字.
offset, length; 获取 idx 在 [offset, offset + length) 范围的成语. 若 length 为 -1, 则
获取 [offset, +无穷大) 范围内的成语列表. 若 offset + length 超出范围, 则视为'+无穷大'.
*/
func (this *IdiomGraph) BeginWith(word rune, offset, length int) []*IdiomNode {
	nodes := this.bindex[word]
	start, end := normalize(offset, length, len(nodes))
	return nodes[start:end]
}

// 获取以 word 结尾的成语列表.
func (this *IdiomGraph) EndWith(word rune, offset, length int) []*IdiomNode {
	nodes := this.eindex[word]
	start, end := normalize(offset, length, len(nodes))
	return nodes[start:end]
}

// 获取以 bword 开头, 以 eword 结尾的成语列表.
func (this *IdiomGraph) BeginEndWith(bword, eword rune, offset, length int) []*IdiomNode {
	nodes := this.beindex[beIndexKey{bword, eword}]
	start, end := normalize(offset, length, len(nodes))
	return nodes[start:end]
}

// 获取从 b 到 e 的最短路径. 若不存在则 len(RETURN) == 0. 否则返回 [b, n1, n2,..., e] 这种形式的结果.
func (this *IdiomGraph) ShortestPath(b, e *IdiomNode) []*IdiomNode {
	path, _, _ := this.shortest_paths.Between(b, e)

	// TODO (pp-qq): 不知道这里 path 的布局如何?
	returnpath := make([]*IdiomNode, 0, len(path))
	for _, path_node := range path {
		returnpath = append(returnpath, path_node.(*IdiomNode))
	}

	return returnpath
}
