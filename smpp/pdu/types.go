// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdu

import "github.com/devyx-tech/go-smpp/smpp/pdu/pdufield"

// PDU Types.
const (
	GenericNACKID         ID = 0x80000000
	BindReceiverID        ID = 0x00000001
	BindReceiverRespID    ID = 0x80000001
	BindTransmitterID     ID = 0x00000002
	BindTransmitterRespID ID = 0x80000002
	QuerySMID             ID = 0x00000003
	QuerySMRespID         ID = 0x80000003
	SubmitSMID            ID = 0x00000004
	SubmitSMRespID        ID = 0x80000004
	DeliverSMID           ID = 0x00000005
	DeliverSMRespID       ID = 0x80000005
	UnbindID              ID = 0x00000006
	UnbindRespID          ID = 0x80000006
	ReplaceSMID           ID = 0x00000007
	ReplaceSMRespID       ID = 0x80000007
	CancelSMID            ID = 0x00000008
	CancelSMRespID        ID = 0x80000008
	BindTransceiverID     ID = 0x00000009
	BindTransceiverRespID ID = 0x80000009
	OutbindID             ID = 0x0000000B
	EnquireLinkID         ID = 0x00000015
	EnquireLinkRespID     ID = 0x80000015
	SubmitMultiID         ID = 0x00000021
	SubmitMultiRespID     ID = 0x80000021
	AlertNotificationID   ID = 0x00000102
	DataSMID              ID = 0x00000103
	DataSMRespID          ID = 0x80000103
)

// GenericNACK PDU.
type GenericNACK struct{ *Codec }

func newGenericNACK(hdr *Header) *Codec {
	return &Codec{h: hdr}
}

// NewGenericNACK creates and initializes a GenericNACK PDU.
func NewGenericNACK() Body {
	b := newGenericNACK(&Header{ID: GenericNACKID})
	b.init()
	return b
}

// Bind PDU.
type Bind struct{ *Codec }

func newBind(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.SystemID,
			pdufield.Password,
			pdufield.SystemType,
			pdufield.InterfaceVersion,
			pdufield.AddrTON,
			pdufield.AddrNPI,
			pdufield.AddressRange,
		}}
}

// NewBindReceiver creates a new Bind PDU.
func NewBindReceiver() Body {
	b := newBind(&Header{ID: BindReceiverID})
	b.init()
	return b
}

// NewBindTransceiver creates a new Bind PDU.
func NewBindTransceiver() Body {
	b := newBind(&Header{ID: BindTransceiverID})
	b.init()
	return b
}

// NewBindTransmitter creates a new Bind PDU.
func NewBindTransmitter() Body {
	b := newBind(&Header{ID: BindTransmitterID})
	b.init()
	return b
}

// BindResp PDU.
type BindResp struct{ *Codec }

func newBindResp(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{pdufield.SystemID},
	}
}

// NewBindReceiverResp creates and initializes a new BindResp PDU.
func NewBindReceiverResp() Body {
	b := newBindResp(&Header{ID: BindReceiverRespID})
	b.init()
	return b
}

// NewBindReceiverRespSeq creates and initializes a new BindResp PDU.
func NewBindReceiverRespSeq(seq uint32) Body {
	b := newBindResp(&Header{ID: BindReceiverRespID, Seq: seq})
	b.init()
	return b
}

// NewBindTransceiverResp creates and initializes a new BindResp PDU.
func NewBindTransceiverResp() Body {
	b := newBindResp(&Header{ID: BindTransceiverRespID})
	b.init()
	return b
}

// NewBindTransceiverRespSeq creates and initializes a new BindResp PDU.
func NewBindTransceiverRespSeq(seq uint32) Body {
	b := newBindResp(&Header{ID: BindTransceiverRespID, Seq: seq})
	b.init()
	return b
}

// NewBindTransmitterResp creates and initializes a new BindResp PDU.
func NewBindTransmitterResp() Body {
	b := newBindResp(&Header{ID: BindTransmitterRespID})
	b.init()
	return b
}

// NewBindTransmitterRespSeq creates and initializes a new BindResp PDU.
func NewBindTransmitterRespSeq(seq uint32) Body {
	b := newBindResp(&Header{ID: BindTransmitterRespID, Seq: seq})
	b.init()
	return b
}

// QuerySM PDU.
type QuerySM struct{ *Codec }

func newQuerySM(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
		},
	}
}

// NewQuerySM creates and initializes a new QuerySM PDU.
func NewQuerySM() Body {
	b := newQuerySM(&Header{ID: QuerySMID})
	b.init()
	return b
}

// QuerySMResp PDU.
type QuerySMResp struct{ *Codec }

func newQuerySMResp(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.FinalDate,
			pdufield.MessageState,
			pdufield.ErrorCode,
		},
	}
}

// NewQuerySMResp creates and initializes a new QuerySMResp PDU.
func NewQuerySMResp() Body {
	b := newQuerySMResp(&Header{ID: QuerySMRespID})
	b.init()
	return b
}

// NewQuerySMRespSeq creates and initializes a new QuerySMResp PDU.
func NewQuerySMRespSeq(seq uint32) Body {
	b := newQuerySMResp(&Header{ID: QuerySMRespID, Seq: seq})
	b.init()
	return b
}

// SubmitSM PDU.
type SubmitSM struct{ *Codec }

func newSubmitSM(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.DestAddrTON,
			pdufield.DestAddrNPI,
			pdufield.DestinationAddr,
			pdufield.ESMClass,
			pdufield.ProtocolID,
			pdufield.PriorityFlag,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.ReplaceIfPresentFlag,
			pdufield.DataCoding,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.UDHLength,
			pdufield.GSMUserData,
			pdufield.ShortMessage,
		},
	}
}

// NewSubmitSM creates and initializes a new SubmitSM PDU.
func NewSubmitSM() Body {
	b := newSubmitSM(&Header{ID: SubmitSMID})
	b.init()
	return b
}

// SubmitSMResp PDU.
type SubmitSMResp struct{ *Codec }

func newSubmitSMResp(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
		},
	}
}

// NewSubmitSMResp creates and initializes a new SubmitSMResp PDU.
func NewSubmitSMResp() Body {
	b := newSubmitSMResp(&Header{ID: SubmitSMRespID})
	b.init()
	return b
}

// NewSubmitSMRespSeq creates and initializes a new SubmitSMResp PDU.
func NewSubmitSMRespSeq(seq uint32) Body {
	b := newSubmitSMResp(&Header{ID: SubmitSMRespID, Seq: seq})
	b.init()
	return b
}

// SubmitMulti PDU.
type SubmitMulti struct{ *Codec }

func newSubmitMulti(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.NumberDests,
			pdufield.DestinationList, // contains DestFlag, DestAddrTON and DestAddrNPI for each address
			pdufield.ESMClass,
			pdufield.ProtocolID,
			pdufield.PriorityFlag,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.ReplaceIfPresentFlag,
			pdufield.DataCoding,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.ShortMessage,
		},
	}
}

// NewSubmitMulti creates and initializes a new SubmitMulti PDU.
func NewSubmitMulti() Body {
	b := newSubmitMulti(&Header{ID: SubmitMultiID})
	b.init()
	return b
}

// SubmitMultiResp PDU.
type SubmitMultiResp struct{ *Codec }

func newSubmitMultiResp(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.NoUnsuccess,
			pdufield.UnsuccessSme,
		},
	}
}

// NewSubmitMultiResp creates and initializes a new SubmitMultiResp PDU.
func NewSubmitMultiResp() Body {
	b := newSubmitMultiResp(&Header{ID: SubmitMultiRespID})
	b.init()
	return b
}

// NewSubmitMultiRespSeq creates and initializes a new SubmitMultiResp PDU.
func NewSubmitMultiRespSeq(seq uint32) Body {
	b := newSubmitMultiResp(&Header{ID: SubmitMultiRespID, Seq: seq})
	b.init()
	return b
}

// DeliverSM PDU.
type DeliverSM struct{ *Codec }

func newDeliverSM(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.DestAddrTON,
			pdufield.DestAddrNPI,
			pdufield.DestinationAddr,
			pdufield.ESMClass,
			pdufield.ProtocolID,
			pdufield.PriorityFlag,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.ReplaceIfPresentFlag,
			pdufield.DataCoding,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.UDHLength,
			pdufield.GSMUserData,
			pdufield.ShortMessage,
		},
	}
}

// NewDeliverSM creates and initializes a new DeliverSM PDU.
func NewDeliverSM() Body {
	b := newDeliverSM(&Header{ID: DeliverSMID})
	b.init()
	return b
}

// DeliverSMResp PDU.
type DeliverSMResp struct{ *Codec }

func newDeliverSMResp(hdr *Header) *Codec {
	return &Codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
		},
	}
}

// NewDeliverSMResp creates and initializes a new DeliverSMResp PDU.
func NewDeliverSMResp() Body {
	b := newDeliverSMResp(&Header{ID: DeliverSMRespID})
	b.init()
	return b
}

// NewDeliverSMRespSeq creates and initializes a new DeliverSMResp PDU for a specific seq.
func NewDeliverSMRespSeq(seq uint32) Body {
	b := newDeliverSMResp(&Header{ID: DeliverSMRespID, Seq: seq})
	b.init()
	return b
}

// Unbind PDU.
type Unbind struct{ *Codec }

func newUnbind(hdr *Header) *Codec {
	return &Codec{h: hdr}
}

// NewUnbind creates and initializes a Unbind PDU.
func NewUnbind() Body {
	b := newUnbind(&Header{ID: UnbindID})
	b.init()
	return b
}

// UnbindResp PDU.
type UnbindResp struct{ *Codec }

func newUnbindResp(hdr *Header) *Codec {
	return &Codec{h: hdr}
}

// NewUnbindResp creates and initializes a UnbindResp PDU.
func NewUnbindResp() Body {
	b := newUnbindResp(&Header{ID: UnbindRespID})
	b.init()
	return b
}

// NewUnbindRespSeq creates and initializes a UnbindResp PDU.
func NewUnbindRespSeq(seq uint32) Body {
	b := newUnbindResp(&Header{ID: UnbindRespID, Seq: seq})
	b.init()
	return b
}

// EnquireLink PDU.
type EnquireLink struct{ *Codec }

func newEnquireLink(hdr *Header) *Codec {
	return &Codec{h: hdr}
}

// NewEnquireLink creates and initializes a EnquireLink PDU.
func NewEnquireLink() Body {
	b := newEnquireLink(&Header{ID: EnquireLinkID})
	b.init()
	return b
}

// EnquireLinkResp PDU.
type EnquireLinkResp struct{ *Codec }

func newEnquireLinkResp(hdr *Header) *Codec {
	return &Codec{h: hdr}
}

// NewEnquireLinkResp creates and initializes a EnquireLinkResp PDU.
func NewEnquireLinkResp() Body {
	b := newEnquireLinkResp(&Header{ID: EnquireLinkRespID})
	b.init()
	return b
}

// NewEnquireLinkRespSeq creates and initializes a EnquireLinkResp PDU for a specific seq.
func NewEnquireLinkRespSeq(seq uint32) Body {
	b := newEnquireLinkResp(&Header{ID: EnquireLinkRespID, Seq: seq})
	b.init()
	return b
}
