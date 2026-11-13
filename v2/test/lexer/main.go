package main

import "fmt"

func main() {

	var n rune
	for n = 0; n < 1000; n++ {
		fmt.Printf("%U %c\n", n, CharClass(n))
	}

}

func CharClass(c rune) rune {

	if c == 0x20 || c == 0x09 {
		return 's'
	}

	if c == 10 || c == 13 {
		return 'b'
	}

	if c < 32 || (c >= 0x7f && c <= 0x9f) {
		return 'c'
	}
	return 'w'
}

// Check https://chatgpt.com/c/6990d08d-b208-8332-8217-b3372e06db9f
