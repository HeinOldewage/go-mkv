package webm

import (
	"errors"
	"io"
)

var (
	ErrParse         = errors.New("Parse error")
	ErrUnexpectedEOF = errors.New("Unexpected EOF")
)

// InitDocument creates a WebM document containing the file data
// It does not do any parsing
func InitDocument(data []byte) *Document {
	doc := new(Document)

	doc.Data = data
	doc.Length = uint64(len(data))
	doc.Cursor = 0

	return doc
}

// ParseAll parses the entire WebM document
// When an EBML/WebM element is encountered, it calls the provided function
// and passes the newly parsed element
func (doc *Document) ParseAll(c func(Element)) error {
	for doc.Cursor < doc.Length {
		// TODO: Get the element's level
		el, err := doc.ParseElement()
		if err != nil {
			return err
		}

		c(el)
	}

	return nil
}

// ParseElement parses an EBML/WebM element starting at the document's current cursor position
// Because of its nature, it does not set the elements's parent or level.
func (doc *Document) ParseElement() (Element, error) {
	var el Element
	var s = doc.Cursor

	if doc.Cursor >= doc.Length {
		return el, io.EOF
	}

	id, err := doc.GetElementID()
	if err != nil {
		return el, err
	}

	size, err := doc.GetElementSize()
	if err != nil {
		return el, err
	}

	el = Element{
		GetElementRegister(id),
		nil,
		0,
		s,
		size,
		nil,
		nil,
	}

	if doc.Cursor+size < doc.Length {
		el.Bytes = doc.Data[s : doc.Cursor+size]
	}

	if el.Type != ElementTypeMaster {
		if doc.Cursor+size > doc.Length {
			return el, ErrUnexpectedEOF
		}

		el.Content = doc.Data[doc.Cursor : doc.Cursor+size]
		doc.Cursor += el.Size
	}

	return el, nil
}

// GetElementID tries to parse the next element's id,
// starting from the document's current cursor position.
func (doc *Document) GetElementID() (uint32, error) {
	if doc.Cursor >= doc.Length {
		return 0, io.EOF
	}

	var s = doc.Cursor
	var b = doc.Data[doc.Cursor]

	if ((b & 0x80) >> 7) == 1 { // Class A ID (on 1 byte)
		doc.Cursor++
		return uint32(b), nil
	}
	if ((b & 0x40) >> 6) == 1 { // Class B ID (on 2 byte)
		doc.Cursor += 2
		return uint32(pack(2, doc.Data[s:doc.Cursor])), nil
	}
	if ((b & 0x20) >> 5) == 1 { // Class C ID (on 3 bytes)
		doc.Cursor += 3
		return uint32(pack(3, doc.Data[s:doc.Cursor])), nil
	}
	if ((b & 0x10) >> 4) == 1 { // Class D ID (on 4 bytes)
		doc.Cursor += 4
		return uint32(pack(4, doc.Data[s:doc.Cursor])), nil
	}

	return 0, ErrParse
}

// GetElementSize tries to parse the next element's size,
// starting from the document's current cursor position.
func (doc *Document) GetElementSize() (uint64, error) {
	if doc.Cursor >= doc.Length {
		return 0, io.EOF
	}

	var mask byte
	var length uint64
	var b = doc.Data[doc.Cursor]

	if b >= 0x80 {
		length = 1
		mask = 0x7f
	} else if b >= 0x40 {
		length = 2
		mask = 0x3f
	} else if b >= 0x20 {
		length = 3
		mask = 0x1f
	} else if b >= 0x10 {
		length = 4
		mask = 0x0f
	} else if b >= 0x08 {
		length = 5
		mask = 0x07
	} else if b >= 0x04 {
		length = 6
		mask = 0x03
	} else if b >= 0x02 {
		length = 7
		mask = 0x01
	} else if b >= 0x01 {
		length = 8
		mask = 0x00
	} else {
		return 0, ErrParse
	}

	if doc.Cursor+length >= doc.Length {
		return 0, ErrUnexpectedEOF
	}

	var v uint64 = 0
	var s = doc.Cursor

	for i, l := doc.Cursor, length; i < s+l; i++ {
		bb := doc.Data[i]

		if i == s {
			bb &= mask
		}

		v <<= 8
		v += uint64(bb)

		doc.Cursor++
	}

	return v, nil
}
