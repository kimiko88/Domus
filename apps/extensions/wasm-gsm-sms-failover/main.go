// Package main implements AT command modem control and SMS failover dispatching.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

type SMSDispatch struct {
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

type ModemStatus struct {
	SignalQualityCSQ int    `json:"signal_quality_csq"` // 0-31
	Registered       bool   `json:"registered"`
	OperatorName     string `json:"operator_name"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_build_at_sms_command
func domus_build_at_sms_command(ptr unsafe.Pointer, size int) uint64 {
	buf := unsafe.Slice((*byte)(ptr), size)
	var dispatch SMSDispatch
	if err := json.Unmarshal(buf, &dispatch); err != nil {
		return 0
	}

	atCommand := fmt.Sprintf("AT+CMGS=\"%s\"\r%s\x1A", dispatch.Recipient, dispatch.Message)
	outBytes := []byte(atCommand)

	outPtr := uintptr(unsafe.Pointer(&outBytes[0]))
	return (uint64(outPtr) << 32) | uint64(len(outBytes))
}

func main() {}
