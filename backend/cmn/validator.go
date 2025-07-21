package cmn

import "github.com/go-playground/validator/v10"

var validate *validator.Validate = validator.New()

// Validate 校验
func Validate(instance interface{}) error {
	err := validate.Struct(instance)
	if err != nil {
		z.Error("validation fail:" + err.Error())
	}
	return err
}
