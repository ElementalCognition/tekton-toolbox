package pipelineresolver

type ExprInvalidError struct {
	Err error
}

func (e *ExprInvalidError) Error() string {
	return e.Err.Error()
}
