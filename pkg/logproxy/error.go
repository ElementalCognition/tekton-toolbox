package logproxy

type BucketNotExistError struct {
	Err error
}

var _ error = (*BucketNotExistError)(nil)

func (e *BucketNotExistError) Error() string {
	return e.Err.Error()
}

type ObjectNotExistError struct {
	Err error
}

var _ error = (*ObjectNotExistError)(nil)

func (e *ObjectNotExistError) Error() string {
	return e.Err.Error()
}
