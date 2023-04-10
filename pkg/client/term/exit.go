package term

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExitCode struct {
	Code int
	Err  error
}

func ToExitCode(conn io.ReadCloser) ExitCode {
	defer conn.Close()

	status := metav1.Status{}
	data, err := io.ReadAll(conn)
	if err != nil {
		return ExitCode{
			Code: 1,
			Err:  err,
		}
	}
	if len(data) == 0 {
		return ExitCode{}
	}

	err = json.Unmarshal(data, &status)
	if err != nil {
		return ExitCode{
			Code: 1,
			Err:  err,
		}
	}

	if status.Status == "Success" {
		return ExitCode{
			Code: 0,
		}
	}

	if status.Reason == "NonZeroExitCode" && status.Details != nil {
		for _, cause := range status.Details.Causes {
			if cause.Type == "ExitCode" {
				i, err := strconv.Atoi(cause.Message)
				if err == nil {
					return ExitCode{
						Code: i,
					}
				}
			}
		}
	} else if status.Reason == "InternalError" && status.Details != nil {
		for _, cause := range status.Details.Causes {
			if cause.Message != "" {
				return ExitCode{
					Code: 1,
					Err:  errors.New(cause.Message),
				}
			}
		}
	}

	return ExitCode{
		Code: 1,
	}
}
