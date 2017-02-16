// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpupower

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func readInt(path string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func writeInt(path string, val int) error {
	return ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", val)), 0)
}
