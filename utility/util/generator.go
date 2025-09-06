package util

import (
	"math/rand"

	"github.com/bwmarrin/snowflake"
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
