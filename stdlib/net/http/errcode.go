// SPDX-FileCopyrightText: © 2024 Kevin Conway
// SPDX-FileCopyrightText: © 2017 Atlassian Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package http

import "fmt"

// errStatusCode is a private error type used to adapt between HTTP protocol
// errors and error rate calculation in the load shedder.
type errStatusCode struct {
	errCode int
}

func (c *errStatusCode) Error() string {
	return fmt.Sprintf("error code is %d", c.errCode)
}
