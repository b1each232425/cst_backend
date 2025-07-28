package time_sync

type MockRepo struct {
	QueryActualExamEndTimeFunc func(examineeId int64) (int64, error)
}

func (m *MockRepo) QueryActualExamEndTime(examineeId int64) (int64, error) {
	return m.QueryActualExamEndTimeFunc(examineeId)
}
