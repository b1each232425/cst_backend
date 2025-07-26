package cmn

var (
	ResetExamEndTimeChan    = make(chan int64, 1000)
	ResetExamEndTimeErrChan = make(chan error, 1000)
)
