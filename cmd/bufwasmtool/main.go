// Copyright 2020-2023 Buf Technologies, Inc.
//
// All rights reserved.

package main

import (
	"os"
	"path"

	"github.com/bufbuild/plugins/bufwasmtool"
)

func main() {
	bufwasmtool.Main(path.Base(os.Args[0]))
}
