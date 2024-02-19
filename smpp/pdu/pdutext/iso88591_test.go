// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

import (
	"bytes"
	"io/ioutil"
	"testing"
)

const (
	iso88591TypeCode = 0x00
)

var (
	iso88591Bytes     = readBytesFromFileISO88591(testDataDir + "/iso88591_test.txt")
	iso88591UTF8Bytes = readBytesFromFileISO88591(testDataDir + "/iso88591_test_utf8.txt")
)

func TestISO88591Encoder(t *testing.T) {
	want := []byte(iso88591Bytes)
	text := []byte(iso88591UTF8Bytes)
	s := ISO88591(text)
	if s.Type() != iso88591TypeCode {
		t.Fatalf("Unexpected data type; want %d, have %d", iso88591TypeCode, s.Type())
	}
	have := s.Encode()
	if !bytes.Equal(want, have) {
		t.Fatalf("Unexpected text; want %q, have %q", want, have)
	}
}

func TestISO88591Decoder(t *testing.T) {
	want := []byte(iso88591UTF8Bytes)
	text := []byte(iso88591Bytes)
	s := ISO88591(text)
	if s.Type() != iso88591TypeCode {
		t.Fatalf("Unexpected data type; want %d, have %d", iso88591TypeCode, s.Type())
	}
	have := s.Decode()
	if !bytes.Equal(want, have) {
		t.Fatalf("Unexpected text; want %q, have %q", want, have)
	}
}

func readBytesFromFileISO88591(filename string) []byte {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		panic("Error reading testdata file; " + filename + ", err " + err.Error())
	} else {
		return dat
	}
}
