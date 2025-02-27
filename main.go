package main

import (
	"walkable/src/bsky"
	"walkable/src/utils"
)

func main() {
	p := utils.CreatePost()
	bsky.Run(p)
}
