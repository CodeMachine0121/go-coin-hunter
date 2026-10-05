package domains

import "errors"

var (
	ErrPipelineRunNotFound  = errors.New("找不到這個輪次")
	ErrInvalidPipelineRunID = errors.New("輪次編號必須是正整數")
)
