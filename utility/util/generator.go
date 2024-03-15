package util

import (
	"github.com/bwmarrin/snowflake"
	"math/rand"
)

var node *snowflake.Node

func init() {

	var err error
	if node, err = snowflake.NewNode(rand.Int63n(1023)); err != nil {
		panic(err)
	}
}

func GenerateId() string {
	return node.Generate().String()
}
