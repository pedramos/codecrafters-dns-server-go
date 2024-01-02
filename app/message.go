package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var endian = binary.BigEndian

func bool2int(b bool) uint16 {
	if b {
		return 1
	}
	return 0
}

func int2bool(i uint16) bool {
	if i == 1 {
		return true
	}
	return false
}

func bits2Uint(bits []bool) uint16 {
	var v uint16
	for _, b := range bits {
		v <<= 1
		if b {
			v |= 1
		}
	}
	return v
}

func uint2Bits(v uint16, numBits int) []bool {
	b := make([]bool, 16)
	for i := 0; v != 0; i++ {
		b[15-i] = int2bool(1 & v)
		v = v >> 1
	}
	return b[16-numBits:]
}

type Message struct {
	h   Header
	q   []Question
	ans Answer
	// Authority
	// some padding I guess
}

func DecodeMessage(input []byte) (Message, error) {
	var (
		m   = Message{}
		err error

		offset int
	)

	// fmt.Printf("header=%d\n", len(input))

	m.h, offset, err = DecodeHeader(input)
	if err != nil {
		return m, fmt.Errorf("parsing message: %v", err)
	}
	// fmt.Printf("question=%d\n", len(input))
	m.q, offset, err = DecodeQuestion(input, offset, m.h)
	if err != nil {
		return m, fmt.Errorf("parsing message: %v", err)
	}
	return m, nil
}

func (m *Message) Reply() error {
	m.h.QR = int2bool(1)
	m.h.ANCount = 1

	if m.h.OpCode == [4]bool{int2bool(0), int2bool(0), int2bool(0), int2bool(0)} {
		copy(m.h.RCode[:], uint2Bits(0, 4))
	} else {
		copy(m.h.RCode[:], uint2Bits(4, 4))
	}
	m.ans = AnswerQuestion(m.q)
	return nil
}

func (m Message) Encode() []byte {
	var buff = new(bytes.Buffer)

	buff.Write(m.h.Encode())
	buff.Write(m.q.Encode())
	buff.Write(m.ans.Encode())

	//fmt.Printf("%#v\n", buff.Bytes())
	return buff.Bytes()
}

type Labels []string

func DecodeLabels(input []byte, offset int) (Labels, int, error) {
	var (
		r      = bytes.NewReader(input)
		buff   = new(bytes.Buffer)
		sb     strings.Builder
		labels Labels
	)

	input = input[offset:]

	for n > 0 {
		b, err := r.ReadByte()
		if err != nil {
			return Labels{}, offset, fmt.Errorf("reading and parsing label: %v", err)
		}
		switch {
		case b == '\x00':
			labels = append(labels, sb.String())
			sb.Reset()
			n--
			continue
		// is a pointer
		case b>>6 == 3:
			ptr := int(binary.BigEndian.Uint16(input) & 0x3FFF)
			label := parseLabel(bytes.NewReader(input[ptr:]))
			sb.Write(label)
		case b>>6 == 0:
			r.UnreadByte()
			label := parseLabel(r)
		}

		if sb.Len() != 0 {
			sb.WriteRune('.')
		}
		sz := uint8(b)

		buff.Reset()
		for i := uint8(0); i < sz; i++ {
			b, err = r.ReadByte()
			if err != nil {
				return Labels{}, offset, fmt.Errorf("reading and parsing label: %v", err)
			}
			buff.WriteByte(b)
		}
		sb.Write(buff.Bytes())
	}
	labelSz := len(input) - r.Len()

	return labels, labelSz, nil
}

func parseLabel(io.Reader) Label {

}

func (labels Labels) Encode() []byte {
	var buff = new(bytes.Buffer)
	for _, l := range labels {
		labels := strings.Split(l, ".")
		for _, label := range labels {
			binary.Write(buff, endian, uint8(len(label)))
			buff.Write([]byte(label))
		}
		buff.WriteByte('\x00')
	}
	return buff.Bytes()
}

type Answer struct {
	//The domain name encoded as a sequence of labels.
	Name Labels
	//1 for an A record, 5 for a CNAME record etc., full list here
	Type uint16
	//Usually set to 1 (full list here)
	Class uint16
	//The duration in seconds a record can be cached before requerying. (Time-To-Live)
	TTL uint32
	//Length of the RDATA field in bytes.
	RDLength uint16
	//Data specific to the record type (RDATA)
	RData []byte
}

func AnswerQuestion(q Question) Answer {
	return Answer{
		Name:     q.Name,
		Type:     1,
		Class:    1,
		TTL:      60,
		RDLength: 4,
		RData:    []byte{'\x08', '\x08', '\x08', '\x08'},
	}
}

func (ans Answer) Encode() []byte {
	var buff = new(bytes.Buffer)

	buff.Write(ans.Name.Encode())

	binary.Write(buff, endian, ans.Type)
	binary.Write(buff, endian, ans.Class)
	binary.Write(buff, endian, ans.TTL)
	binary.Write(buff, endian, ans.RDLength)

	buff.Write(ans.RData)
	return buff.Bytes()
}

type Question struct {
	// Domain names in DNS packets are encoded as a sequence of labels.
	//
	// Labels are encoded as <length><content>, where <length> is a single byte that specifies
	// the length of the label, and <content> is the actual content of the label. The sequence of
	// labels is terminated by a null byte (\x00).
	Name Labels

	Type  uint16
	Class uint16
}

func DecodeQuestion(input []byte, offset int, h Header) (Question, int, error) {
	var (
		q   Question
		err error
	)

	input = input[offset:]

	q.Name, offset, err = DecodeLabels(input, offset, h.QDCount)
	if err != nil {
		return q, offset, fmt.Errorf("reading and parsing question section: %v", err)
	}
	q.Type = endian.Uint16(input)
	input = input[2:]
	offset += 2

	q.Class = endian.Uint16(input)
	offset += 2

	return q, offset, nil
}

func (q Question) Encode() []byte {

	var buff = new(bytes.Buffer)

	buff.Write(q.Name.Encode())

	binary.Write(buff, endian, q.Type)
	binary.Write(buff, endian, q.Class)
	return buff.Bytes()
}

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

func DesiredHeader() Header {
	return Header{
		PackageID: 1234,
		QR:        int2bool(1),
		QDCount:   1,
		ANCount:   1,
	}
}

func DecodeHeader(input []byte) (Header, int, error) {
	var (
		h      Header
		mask   uint64
		mask16 uint16
		tmp    uint16
		t      uint16

		offset = 0
	)

	h.PackageID = endian.Uint16(input)
	input = input[2:]
	offset += 2

	//  QR / OpCode / AA / TC / RD

	bitsBlock := endian.Uint16(input)
	tmp = bitsBlock >> 8
	input = input[2:]
	offset += 2

	h.QR = int2bool(tmp >> 7)

	mask, _ = strconv.ParseUint("01111000", 2, 16)
	mask16 = uint16(mask)
	t = (tmp & mask16) >> 3
	for i := 0; i < 4; i++ {
		h.OpCode[3-i] = int2bool((t >> i) & 1)
	}

	mask, _ = strconv.ParseUint("100", 2, 16)
	mask16 = uint16(mask)
	h.AA = int2bool((tmp & mask16) >> 2)

	mask, _ = strconv.ParseUint("10", 2, 16)
	mask16 = uint16(mask)
	h.TC = int2bool((tmp & mask16) >> 1)

	mask, _ = strconv.ParseUint("1", 2, 16)
	mask16 = uint16(mask)
	h.RD = int2bool(tmp & mask16)

	// RA / Z / RCode

	mask, _ = strconv.ParseUint("11111111", 2, 16)
	mask16 = uint16(mask)
	tmp = bitsBlock & mask16

	h.RA = int2bool(tmp >> 7)

	mask, _ = strconv.ParseUint("01110000", 2, 16)
	mask16 = uint16(mask)
	t = (tmp & mask16) >> 4
	for i := 0; i < 3; i++ {
		h.Z[2-i] = int2bool((t >> i) & 1)
	}

	mask, _ = strconv.ParseUint("00001111", 2, 16)
	mask16 = uint16(mask)
	t = tmp & mask16
	//fmt.Printf("got=%#v\n", strconv.FormatInt(int64(t), 2))
	for i := 0; i < 4; i++ {
		h.RCode[3-i] = int2bool((t << i) & 1)
	}
	//fmt.Printf("%#v\n", h.RCode)
	h.QDCount = endian.Uint16(input)
	input = input[2:]
	h.ANCount = endian.Uint16(input)
	input = input[2:]
	h.NSCount = endian.Uint16(input)
	input = input[2:]
	h.ARCount = endian.Uint16(input)
	input = input[2:]

	offset += 8

	return h, offset, nil
}

func (h Header) Encode() []byte {

	var buff = new(bytes.Buffer)

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

	return buff.Bytes()
}
