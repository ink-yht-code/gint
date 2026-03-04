package errs

// ServiceID 服务 ID
const ServiceID = 101

// 业务码 = ServiceID * 10000 + BizCode
const (
	CodeSuccess       = ServiceID * 10000 + 0
	CodeInvalidParam  = ServiceID * 10000 + 1
	CodeUnauthorized  = ServiceID * 10000 + 2
	CodeForbidden     = ServiceID * 10000 + 3
	CodeNotFound      = ServiceID * 10000 + 4
	CodeConflict      = ServiceID * 10000 + 5
	CodeInternalError = ServiceID * 10000 + 9999
)
