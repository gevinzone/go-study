package web

import (
	"fmt"
	"regexp"
	"strings"
)

type router struct {
	// trees 是按照 HTTP 方法来组织的
	// 如 GET => *node
	trees map[string]*node
}

func newRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// addRoute 注册路由。
// method 是 HTTP 方法
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 不能在同一个位置同时注册通配符路由和参数路由，例如 /user/:id 和 /user/* 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
func (r *router) addRoute(method string, path string, handler HandleFunc) {
	_ = r.checkPathFormat(path)

	root, isRootPath := r.handleRootRouter(method, path, handler)
	if isRootPath {
		return
	}
	r.handleSegmentRouter(root, method, path, handler)
}

func (r *router) checkPathFormat(path string) bool {
	if path == "" {
		panic("web: 路由是空字符串")
	}
	if path[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}

	if path != "/" && path[len(path)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}
	return true
}

// 返回root节点，以及请求path是否为root
func (r *router) handleRootRouter(method, path string, handler HandleFunc) (*node, bool) {
	root, ok := r.trees[method]
	// 这是一个全新的 HTTP 方法，创建根节点
	if !ok {
		// 创建根节点
		root = &node{path: "/"}
		r.trees[method] = root
	}
	if path == "/" {
		if root.handler != nil {
			panic("web: 路由冲突[/]")
		}
		root.handler = handler
	}
	return root, path == "/"
}

func (r *router) handleSegmentRouter(n *node, method, path string, handler HandleFunc) {
	segs := strings.Split(path[1:], "/")
	// 开始一段段处理
	for _, s := range segs {
		if s == "" {
			panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", path))
		}
		n = n.childOrCreate(s)
	}
	if n.handler != nil && n.path != "*" {
		panic(fmt.Sprintf("web: 路由冲突[%s]", path))
	}
	n.handler = handler
}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	if path == "/" {
		return &matchInfo{n: root}, true
	}

	return r.findPathRoute(root, path)
}

func (r *router) findPathRoute(curNode *node, path string) (*matchInfo, bool) {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	mi := &matchInfo{}
	var prev *node
	for _, s := range segs {
		var matchParam, ok bool
		prev = curNode
		curNode, matchParam, ok = curNode.childOf(s)
		if !ok {
			if prev.typ == nodeTypeAny {
				mi.n = prev
				return mi, true
			}
			return nil, false
		}
		if matchParam {
			mi.addValue(curNode.paramName, s)
		}
	}
	mi.n = curNode
	return mi, true
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic = iota
	// 正则路由
	nodeTypeReg
	// 路径参数路由
	nodeTypeParam
	// 通配符路由
	nodeTypeAny
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 路径参数匹配：形式 :param_name
// 3. 通配符匹配：*
// 这是不回溯匹配
type node struct {
	typ nodeType

	path string
	// children 子节点
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler HandleFunc

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	paramChild *node
	// 正则路由和参数路由都会使用这个字段
	paramName string

	// 正则表达式
	regChild *node
	regExpr  *regexp.Regexp
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否是命中参数路由
// 第三个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool, bool) {
	if n.children == nil {
		if n.regChild != nil {
			matched := n.regChild.regExpr.MatchString(path)
			return n.regChild, matched, matched
		}
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		//return n.starChild, false, true
		return n.starChild, false, n.starChild != nil
	}
	res, ok := n.children[path]
	if !ok {
		if n.regChild != nil {
			matched := n.regChild.regExpr.MatchString(path)
			return n.regChild, true, matched
		}
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		return n.starChild, false, n.starChild != nil
	}
	return res, false, ok
}

// childOrCreate 查找子节点，
// 首先会判断 path 是不是通配符路径
// 其次判断 path 是不是参数路径，即以 : 开头的路径
// 最后会从 children 里面查找，
// 如果没有找到，那么会创建一个新的节点，并且保存在 node 里面
func (n *node) childOrCreate(path string) *node {
	if path == "*" {
		return n.starChildOrCreate(path)
	}

	paramName, regExpr := n.matchAndParseRegExp(path)
	// 解析到正则，是正则路由
	if regExpr != nil {
		return n.regexChildOrCreate(path, paramName, regExpr)
	}

	// 以 : 开头，我们认为是参数路由
	if path[0] == ':' {
		return n.paramChildOrCreate(path)
	}

	return n.staticChildOrCreate(path)
}

func (n *node) starChildOrCreate(path string) *node {
	_ = n.isStarChildAvailable(path)
	if n.starChild == nil {
		n.starChild = &node{path: path, typ: nodeTypeAny}
	}
	return n.starChild
}

func (n *node) isStarChildAvailable(path string) bool {
	if n.paramChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由 [%s]", path))
	}
	if n.regChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由 [%s]", path))
	}
	return true
}

func (n *node) regexChildOrCreate(path, paramName string, regExpr *regexp.Regexp) *node {
	_ = n.isRegexChildAvailable(path)
	if n.regChild == nil {
		n.regChild = &node{path: path, paramName: paramName, regExpr: regExpr, typ: nodeTypeReg}
	}

	return n.regChild
}

func (n *node) isRegexChildAvailable(path string) bool {
	if n.starChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由 [%s]", path))
	}
	if n.paramChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册正则路由和参数路由 [%s]", path))
	}
	if n.regChild != nil && n.regChild.path != path {
		panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s，新注册 %s", n.regChild.path, path))
	}

	return true
}

func (n *node) paramChildOrCreate(path string) *node {
	_ = n.isParamChildAvailable(path)
	if n.paramChild == nil {
		n.paramChild = &node{path: path, paramName: path[1:], typ: nodeTypeParam}
	}
	return n.paramChild
}

func (n *node) isParamChildAvailable(path string) bool {
	if n.starChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", path))
	}
	if n.regChild != nil {
		panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册正则路由和参数路由 [%s]", path))
	}
	if n.paramChild != nil && n.paramChild.path != path {
		panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s，新注册 %s", n.paramChild.path, path))
	}

	return true
}
func (n *node) staticChildOrCreate(path string) *node {
	if n.children == nil {
		n.children = make(map[string]*node)
	}
	child, ok := n.children[path]
	if !ok {
		child = &node{path: path, typ: nodeTypeStatic}
		n.children[path] = child
	}
	return child
}

// 检查是否满足正则路由的规则
// 满足，则返回参数名和正则表达式
func (n *node) matchAndParseRegExp(path string) (string, *regexp.Regexp) {
	if !strings.HasPrefix(path, ":") || !strings.HasSuffix(path, ")") || !strings.Contains(path, "(") {
		return "", nil
	}
	segs := strings.SplitN(path[1:len(path)-1], "(", 2)

	if reg := regexp.MustCompile(segs[1]); reg == nil {
		return "", nil
	} else {
		return segs[0], reg
	}
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
}

func (m *matchInfo) addValue(key string, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = map[string]string{key: value}
	}
	m.pathParams[key] = value
}
