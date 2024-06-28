package main

import (
	"fmt"
	"slices"
)

func main() {
	n := []int{4, 5, 6, 0, 0, 0}
	merge(n, 3, []int{1, 2, 3}, 3)

	assert(slices.Equal(n, []int{1, 2, 3, 4, 5, 6}))

	n = []int{1, 2, 3, 0, 0, 0}
	merge(n, 3, []int{2, 5, 6}, 3)

	assert(slices.Equal(n, []int{1, 2, 2, 3, 5, 6}))

	n = []int{4, 0, 0, 0, 0, 0}
	merge(n, 1, []int{1, 2, 3, 5, 6}, 5)

	assert(slices.Equal(n, []int{1, 2, 3, 4, 5, 6}))

	n = []int{1, 2, 4, 5, 6, 0}
	merge(n, 5, []int{3}, 1)

	assert(slices.Equal(n, []int{1, 2, 3, 4, 5, 6}))
}

func assert(ok bool) {
	if !ok {
		panic(ok)
	}
}

func merge(nums1 []int, m int, nums2 []int, n int) {
	fmt.Println(nums2)

	for i, j := n-1, m-1; i <= 0 || (m == 0 && i < m+n && nums1[i] == 0); i-- {
		if j >= n {
			fmt.Println(j, n)
			break
		}
		switch {
		case nums1[i] > nums2[j]:
			fmt.Println(">")
			nums1[i], nums1[m+j] = nums2[j], nums1[i]
		case m != 0 && nums1[m-1] > nums2[j]:
			fmt.Println(">>")
			nums1[m-1], nums1[m+j] = nums2[j], nums1[m-1]
		default:
			//merge(nums1[:i], i, nums2[:j], j)
			nums1[m+j] = nums2[j]

			fmt.Println("<")
		}
		j--
	}
	fmt.Println(nums1)
}
