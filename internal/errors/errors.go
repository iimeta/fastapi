package errors

import (
	"errors"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

var (
	ERR_SERVER_ERROR      = gerror.NewCode(gcode.New(-1, "Server errors", nil))
	ERR_PERMISSION_DENIED = gerror.NewCode(gcode.New(999, "无权限访问", nil))
	ERR_INVALID_ACCOUNT   = gerror.NewCode(gcode.New(10001, "Invalid Mobile", nil))
)

func New(text string) error {
	return errors.New(text)
}

func Newf(format string, args ...interface{}) error {
	return gerror.Newf(format, args...)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}
