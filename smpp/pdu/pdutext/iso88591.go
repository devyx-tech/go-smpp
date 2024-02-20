// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

import (
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// ISO88591 text codec.
type ISO88591 []byte

// Type implements the Codec interface.
func (s ISO88591) Type() DataCoding {
	return DefaultType
}

// Encode to ISO88595.
func (s ISO88591) Encode() []byte {
	e := charmap.ISO8859_1.NewEncoder()
	es, _, err := transform.Bytes(e, s)
	if err != nil {
		return s
	}
	return es
}

// Decode from ISO88595.
func (s ISO88591) Decode() []byte {
	e := charmap.ISO8859_1.NewDecoder()
	es, _, err := transform.Bytes(e, s)
	if err != nil {
		return s
	}
	return es
}
