// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"encoding/json"
	"errors"
)

type requestType string

// These constants are acronyms, and are used inside HTTP headers to determine
// the type of request.
const (
	rtLongPollEstablishConnection requestType = "lpec" // long-poll-establish-connection
	rtLongPoll                                = "lp"   // long-poll
	rtMessage                                 = "m"    // message
)

func (r requestType) valid() bool {
	switch r {
	case rtLongPollEstablishConnection:
		return true
	case rtLongPoll:
		return true
	case rtMessage:
		return true
	}
	return false
}

// Special messages for different server events, only intended to be entirely
// unique.
var (
	connect int

	// Special message for when an client connection is made.
	Connect = &connect
)

// Two types of request are sent from Organics
//
// 1.
//     Request (Sent to the other connection end to 'Request' something)
//     Looks like: [id, requestName, args]
//
//     Where id is an float64 (JSON 'Number' type) uniquely representing this
//     request to the person whom sent it, where requestName is any valid JSON
//     data type, and where args is an JSON Array type, of any number of valid
//     JSON data types.
//
// 2.
//     Response (Sent as an 'Response' to an previously made Request)
//     Looks like: [id, args]
//
//     Where id is an float64 (JSON 'Number' type) uniquely representing this
//     request to the person whom sent it, and where args is an JSON Array
//     type, of any number of valid JSON data types.
//
type message struct {
	id          float64
	requestName interface{}
	args        []interface{}
	isRequest   bool
}

func newRequestMessage(id float64, requestName interface{}, args []interface{}) *message {
	m := &message{}
	m.id = id
	m.requestName = requestName
	m.args = args
	m.isRequest = true
	return m
}

func newResponseMessage(id float64, args []interface{}) *message {
	m := &message{}
	m.id = id
	m.args = args
	m.isRequest = false
	return m
}

// JsonEncode encodes this *message, m, into an JSON-encoded []byte, or returns
// an error if one is encountered.
func (m *message) jsonEncode() (encoded []byte, err error) {
	if m.isRequest {
		encoded, err = json.Marshal([]interface{}{m.id, m.requestName, m.args})
	} else {
		if len(m.args) == 0 {
			encoded, err = json.Marshal([]interface{}{m.id})
		} else {
			encoded, err = json.Marshal([]interface{}{m.id, m.args})
		}
	}
	if err != nil {
		err = errors.New("Error encoding JSON; " + err.Error())
	}
	return
}

// JsonDecode decodes the data parameter, an array of JSON-encoded []byte, into
// this *message, m, or returns an error if one is encountered.
func (m *message) jsonDecode(data []byte) error {
	var decoded []interface{}
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New("Error decoding JSON; " + err.Error())
	}

	var ok bool
	if len(decoded) == 1 {
		// It's an response, in format of [id]
		m.isRequest = false

		m.id, ok = decoded[0].(float64)
		if !ok {
			return errors.New("Error decoding JSON; id is not an json number!")
		}
		m.args = make([]interface{}, 0)

	} else if len(decoded) == 2 {
		// It's an response, in format of [id, args]
		m.isRequest = false

		m.id, ok = decoded[0].(float64)
		if !ok {
			return errors.New("Error decoding JSON; id is not an json number!")
		}

		m.args, ok = decoded[1].([]interface{})
		if !ok {
			return errors.New("Error decoding JSON; args list is not an json array!")
		}

	} else if len(decoded) == 3 {
		// It's an request, in format of [id, requestName, args]
		m.isRequest = true

		m.id, ok = decoded[0].(float64)
		if !ok {
			return errors.New("Error decoding JSON; id is not an json number!")
		}

		m.requestName = decoded[1]

		m.args, ok = decoded[2].([]interface{})
		if !ok {
			return errors.New("Error decoding JSON; args list is not an json array!")
		}
	}
	return nil
}
