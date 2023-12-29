package util

import (
	"crypto/md5"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/grand"
	"math/rand"
	"slices"
	"strings"
	"time"
)

var node *snowflake.Node

func init() {

	var err error
	node, err = snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}

}

func GenerateId() string {
	return node.Generate().String()
}

// 生成数字验证码
func GenValidateCode(length int) string {
	numeric := [10]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	r := len(numeric)
	rand.Seed(time.Now().UnixNano())

	var sb strings.Builder
	for i := 0; i < length; i++ {
		_, _ = fmt.Fprintf(&sb, "%d", numeric[rand.Intn(r)])
	}
	return sb.String()
}

// 生成随机字符串
func Random(length int) string {
	var result []byte
	bytes := []byte("0123456789abcdefghijklmnopqrstuvwxyz")

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}

	return string(result)
}

// 随机生成指定后缀的图片名
func GenImageName(ext string, width, height int) string {
	str := fmt.Sprintf("%d%s", time.Now().Unix(), Random(10))

	return fmt.Sprintf("%x_%dx%d.%s", md5.Sum([]byte(str)), width, height, ext)
}

func GenFileName(ext string) string {
	str := fmt.Sprintf("%d%s", time.Now().Unix(), Random(10))

	return fmt.Sprintf("%x.%s", md5.Sum([]byte(str)), ext)
}

func BoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func NewMsgId() string {
	return fmt.Sprintf("%s_%d", GenerateId(), gtime.Timestamp())
}

func NewKey(id string, length int, prefix ...string) string {

	key := ""
	n := length

	if len(prefix) > 0 {
		n -= len(prefix[0])
		key += prefix[0]
	}

	l := len(id)

	n = (n - l) / l

	for i := 0; i < l; i++ {
		key += strings.Join(slices.Insert(strings.Split(grand.Letters(n), ""), grand.Intn(n), id[i:i+1]), "")
	}

	if len(key) < length {
		key += grand.Letters(length - len(key))
	}

	return key
}
