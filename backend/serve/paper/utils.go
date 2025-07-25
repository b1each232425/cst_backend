package paper

import "github.com/pkg/errors"

func validateIDs(ids []int64) error {
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 0 {
			err := errors.Errorf("ID必须大于0 %d", id)
			z.Error(err.Error())
			return err
		}
		if seen[id] {
			err := errors.Errorf("存在重复ID %d", id)
			z.Error(err.Error())
			return err
		}
		seen[id] = true
	}
	return nil
}
