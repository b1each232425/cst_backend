package cmn

import (
	"fmt"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate = validator.New()

// Validate 校验
func Validate(instance interface{}) error {
	err := validate.Struct(instance)
	if err != nil {
		err = fmt.Errorf("validation failed:%v", err)
		z.Error(err.Error())
	}
	return err
}
