package neogo

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *BufferUpdateArgs) DecodeMsg(dc *msgp.Reader) (err error) {
	var ssz uint32
	ssz, err = dc.ReadArrayHeader()
	if err != nil {
		return
	}
	if ssz != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: ssz}
		return
	}
	{
		var asz uint32
		asz, err = dc.ReadArrayHeader()
		if err != nil {
			return
		}
		if asz != 0 {
			err = msgp.ArrayError{Wanted: 0, Got: asz}
			return
		}
		for xvk := range z.FunctionArgs {
			z.FunctionArgs[xvk], err = dc.ReadInt64()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *BufferUpdateArgs) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(0)
	if err != nil {
		return
	}
	for xvk := range z.FunctionArgs {
		err = en.WriteInt64(z.FunctionArgs[xvk])
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *BufferUpdateArgs) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendArrayHeader(o, 0)
	for xvk := range z.FunctionArgs {
		o = msgp.AppendInt64(o, z.FunctionArgs[xvk])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *BufferUpdateArgs) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var ssz uint32
		ssz, bts, err = msgp.ReadArrayHeaderBytes(bts)
		if err != nil {
			return
		}
		if ssz != 1 {
			err = msgp.ArrayError{Wanted: 1, Got: ssz}
			return
		}
	}
	var asz uint32
	asz, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		return
	}
	if asz != 0 {
		err = msgp.ArrayError{Wanted: 0, Got: asz}
		return
	}
	for xvk := range z.FunctionArgs {
		z.FunctionArgs[xvk], bts, err = msgp.ReadInt64Bytes(bts)
		if err != nil {
			return
		}
	}
	o = bts
	return
}

func (z *BufferUpdateArgs) Msgsize() (s int) {
	s = 1 + msgp.ArrayHeaderSize + (0 * (msgp.Int64Size))
	return
}
