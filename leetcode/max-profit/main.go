package main

import (
	"fmt"
	"sort"
)

func main() {
	for _, p := range [][]int{
		{3, 3, 5, 0, 0, 3, 1, 4},
		{1, 2, 3, 4, 5},
		{7, 6, 4, 3, 1},
	} {
		fmt.Println(maxProfit(p))
	}
}

func countProfit(price int) {

}

type profit struct {
	buy    int
	buyIdx int
}

func maxProfit(prices []int) int {
	var (
		buy      = make([]int, len(prices))
		balances = make([]int, len(prices))
		idexes   = make([]int, len(prices)*2)
	)
	for i := range prices {
		buy[i] = prices[i]
	}
	fmt.Println("buy", buy)

	for i := range prices {
		curPrice := prices[i]
	inner:
		for j, buyPrice := range buy {
			if i == j {
				continue inner
			}
			if curPrice > buyPrice {
				if buyPrice*curPrice > 0 {
					balances[j] += buyPrice * curPrice
					idexes[j] = i
				}
			}
		}
		buy[i] = prices[i]
	}
	sort.Sort(sort.Reverse(sort.IntSlice(balances)))

	fmt.Println("balances", balances)
	return balances[0]
}
