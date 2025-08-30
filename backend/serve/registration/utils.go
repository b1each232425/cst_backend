package registration

import (
	"fmt"
	"w2w.io/cmn"
)

type Map map[string]interface{}

func ValidateRegisterInfo(R *cmn.TRegisterPlan, Rs []int64) error {
	var err error
	if !R.Name.Valid || R.Name.String == "" {
		err = fmt.Errorf("invalid register Name")
		return err
	}
	if !R.StartTime.Valid || R.StartTime.Int64 <= 0 {
		err = fmt.Errorf("invalid register StartTime")
		return err
	}
	if !R.EndTime.Valid || R.EndTime.Int64 <= 0 {
		err = fmt.Errorf("invalid register EndTime")
		return err
	}
	if !R.ReviewEndTime.Valid || R.ReviewEndTime.Int64 <= 0 {
		err = fmt.Errorf("invalid register ReviewEndTime")
		return err
	}
	if !R.MaxNumber.Valid || R.MaxNumber.Int64 < 0 {
		err = fmt.Errorf("invalid register MaxNumber")
		return err
	}
	if !R.Course.Valid || R.Course.String == "" || (R.Course.String != "00" && R.Course.String != "02" && R.Course.String != "04") {
		err = fmt.Errorf("invalid register Course")
		return err
	}
	if !R.ExamPlanLocation.Valid || R.ExamPlanLocation.String == "" {
		err = fmt.Errorf("invalid register ExamPlanLocation")
		return err
	}
	if Rs != nil && len(Rs) > 0 {
		for _, id := range Rs {
			if id <= 0 {
				err = fmt.Errorf("invalid register practiceIDs")
				return err
			}
		}
	}
	return nil
}
