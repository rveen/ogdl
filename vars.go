// Copyright 2012-2018, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import "errors"

const (
	trueStr  = "true"
	falseStr = "false"
)

// Nodes containing these strings are special
const (
	TypeExpression = "!e"
	TypePath       = "!p"
	TypeVariable   = "!v"
	TypeSelector   = "!s"
	TypeIndex      = "!i"
	TypeGroup      = "!g"
	TypeArguments  = "!a"
	TypeTemplate   = "!t"
	TypeString     = "!string"

	TypeIf    = "!if"
	TypeEnd   = "!end"
	TypeElse  = "!else"
	TypeFor   = "!for"
	TypeBreak = "!break"
)

var (
	ErrNotANumber       = errors.New("not a number")
	ErrNotFound         = errors.New("not found")
	ErrIncompatibleType = errors.New("incompatible type")
	ErrNilReceiver      = errors.New("nil function receiver")
	ErrInvalidIndex     = errors.New("invalid index")
	ErrFunctionNoGraph  = errors.New("functions doesn't return *Graph")
	ErrInvalidArgs      = errors.New("invalid arguments or nil receiver")
)
