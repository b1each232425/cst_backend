package paper

import "github.com/pkg/errors"

func validateIDs(ids []int64) error {
	// 检测数组是否为空
	if len(ids) == 0 {
		err := errors.New("ID List cannot be empty")
		z.Error(err.Error())
		return err
	}
	// 检查ID是否大于0并且没有重复
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			err := errors.Errorf("ID must be greater than 0: %d", id)
			z.Error(err.Error())
			return err
		}
		if seen[id] {
			err := errors.Errorf("Duplicate ID found: %d", id)
			z.Error(err.Error())
			return err
		}
		seen[id] = true
	}
	return nil
}
