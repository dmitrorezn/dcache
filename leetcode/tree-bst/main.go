package main

import "fmt"

func main() {

	tree := &TreeNode{
		Val: 4,
		Left: &TreeNode{
			Val: 1,
			Left: &TreeNode{
				Val:  0,
				Left: nil,
			},
			Right: &TreeNode{
				Val:  2,
				Left: nil,
				Right: &TreeNode{
					Val:  3,
					Left: nil,
				},
			},
		},
		Right: &TreeNode{
			Val: 6,
			Left: &TreeNode{
				Val: 1,
				Left: &TreeNode{
					Val:  5,
					Left: nil,
				},
				Right: nil,
			},
			Right: &TreeNode{
				Val: 7,
				Right: &TreeNode{
					Val:  8,
					Left: nil,
				},
			},
		},
	}
	fmt.Println("tree", tree)
	bstToGst2(tree)
	fmt.Println("tree", tree)
}

type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

func (root *TreeNode) String() string {
	res := " L:"
	for l := root.Left; l != nil; l = l.Left {
		res += l.String()
	}
	res += "R:"
	for r := root.Right; r != nil; r = r.Right {
		res += r.String()
	}
	res += "; => "

	return fmt.Sprint(root.Val) + res
}

func bstToGst2(root *TreeNode) *TreeNode {
	var node = root
	nodeSum := 0
	//var stack =make([]*TreeNode,0)
	for node != nil {
		if node.Right == nil {
			nodeSum += node.Val
			node.Val = nodeSum
			node = node.Left
		}
		if node.Right != nil {
			nodeSum += node.Val
			node.Val = nodeSum
			node = node.Right
		}
		if node.Left == nil {
			node.Val = nodeSum
		}
	}

	return root
}

func bstToGst(root *TreeNode) *TreeNode {
	if root == nil {
		return nil
	}
	root.Val = root.Val + sum(root.Val, root.Right)

	prev := root
	for r := root.Right; r != nil; r = r.Right {
		r.Val = r.Val + sum(r.Val, r.Right)
		r.Left = bstToGstLeft(prev, r.Left)
		prev = r
	}
	prev = root
	for l := root.Left; l != nil; l = l.Left {
		l.Val = l.Val + prev.Val + sum(l.Val, l.Right)
		l.Right = bstToGstLeft(prev, l.Right)
		prev = l
	}

	return root
}

func bstToGstLeft(prev, root *TreeNode) *TreeNode {
	if root == nil {
		return nil
	}
	root.Val = root.Val + prev.Val + sum(root.Val, root.Right)

	root.Left = bstToGstLeft(root, root.Left)
	root.Right = bstToGst(root.Right)

	return root
}

func sum(v int, node *TreeNode) int {
	if node == nil {
		return 0
	}
	s := 0
	if node.Val > v {
		s += node.Val
	}

	return s + sum(v, node.Left) + sum(v, node.Right)
}
