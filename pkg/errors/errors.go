// payment-gateway-poc/pkg/errors/errors.go
// payment-gateway-poc/pkg/errors/errors.go
package errors

import "fmt"

type E struct {
	Code    string
	Message string
	Err     error
}

func (e E) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func Wrap(code, msg string, err error) error {
	return E{Code: code, Message: msg, Err: err}
}
