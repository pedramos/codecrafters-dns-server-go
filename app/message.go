package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Message struct {
	Header
	// Question
	// Answer
	// Authority
	// some padding I guess
}

// https://app.codecrafters.io/courses/dns-server/stages/2?repo=6c3bc592-c18b-4ce0-a5b2-cc25174e4fa0
type Header struct {
	// packade ID: A random ID assigned to query packets. Response packets must reply with the same ID.
	PackageID uint16
	// Query/Response Indicator: 1 for a reply packet, 0 for a question packet.
	QR bool
	// Operation Code: Specifies the kind of query in a message.
	OpCode [4]bool
	// Authoritative Answer  1 if the responding server "owns" the domain queried, i.e., it's authoritative.
	AA bool
	// Truncation (TC): 1 if the message is larger than 512 bytes. Always 0 in UDP responses.
	TC bool
	// Recursion Desired (RD): Sender sets this to 1 if the server should recursively resolve this query, 0 otherwise.
	RD bool
	// Recursion Available (RA): Server sets this to 1 to indicate that recursion is available.
	RA bool
	// Reserved (Z) Used by DNSSEC queries. At inception, it was reserved for future use.
	Z [3]bool
	// Response Code (RCODE): Response code indicating the status of the response.
	RCode [4]bool
	// Question Count (QDCOUNT): Number of questions in the Question section.
	QDCount uint16
	// Answer Record Count (ANCOUNT): Number of records in the Answer section.
	ANCount uint16
	// Authority Record Count (NSCOUNT): Number of records in the Authority section.
	NSCount uint16
	// Additional Record Count (ARCOUNT): Number of records in the Additional section.
	ARCount uint16
}

func NewHeader() Header {
	return Header{
		PackageID: 1234,
		QR:        int2bool(1),
	}
}



func (h Header) Encode() ([12]byte, error) {

	var buff *bytes.Buffer = bytes.NewBuffer(make([]byte, 0,12))
	endian := binary.BigEndian

	binary.Write(buff, endian, h.PackageID)

	var tmp byte

	// convert QR, OpCode, AA, TC and RC to single byte
	if h.QR {
		tmp |= 1
	}
	// Size 4
	for _, b := range h.OpCode {
		tmp <<= 1
		if b {
			tmp |= 1
		}
	}
	tmp <<= 1
	if h.AA {
		tmp |= 1
	}
	tmp <<= 1
	if h.TC {
		tmp |= 1
	}
	tmp <<= 1
	if h.RD {
		tmp |= 1
	}

	buff.WriteByte(tmp)

	// convert RA, Z, AA and RCode into single byte
	tmp = 0
	if h.RA {
		tmp |= 1
	}
	for _, b := range h.Z {
		tmp <<= 1
		if b {
			tmp |= 1
		}
	}
	tmp <<= 1
	for _, b := range h.RCode {
		tmp <<= 1
		if b {
			tmp |= 1
		}
	}

	buff.WriteByte(tmp)

	binary.Write(buff, endian, h.QDCount)
	binary.Write(buff, endian, h.ANCount)
	binary.Write(buff, endian, h.NSCount)
	binary.Write(buff, endian, h.ARCount)
	if buff.Len() != 12 {
		return [12]byte{}, fmt.Errorf("Encoding header into bytes resulted in array of %d and 12 is expected", buff.Len())
	}
	var m [12]byte
	copy(m[:], buff.Bytes())
	return m, nil
}

func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}

func int2bool(i int) bool {
	if i == 1 {
		return true
	}
	return false
}