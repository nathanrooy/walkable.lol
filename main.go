package main

import (
	"walkable/src/bsky"
	"walkable/src/utils"
	"walkable/src/x"
)

func main() {
	p := utils.CreatePost()
	bsky.Run(p)
	x.Run(p)
}
