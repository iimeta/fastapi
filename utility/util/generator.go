package util

import (
	"github.com/bwmarrin/snowflake"
)

var node *snowflake.Node

func init() {

	var err error
	if node, err = snowflake.NewNode(1); err != nil {
		panic(err)
	}
}

func GenerateId() string {
	return node.Generate().String()
}
