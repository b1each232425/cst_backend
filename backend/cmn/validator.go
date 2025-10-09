package cmn

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate = validator.New()

// RegisterValidation 注册自定义验证器
func RegisterValidation(tag string, fn validator.Func) error {
	return validate.RegisterValidation(tag, fn)
}

// Validate 校验
func Validate(instance interface{}) error {
	err := validate.Struct(instance)
	if err != nil {
		err = fmt.Errorf("validation failed:%v", err)
		z.Error(err.Error())
	}
	return err
}
