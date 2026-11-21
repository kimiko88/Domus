package handlers

import "testing"

func TestXiaomiCommandForRequestAllowlist(t *testing.T) {
	cases := []struct {
		name string
		req xiaomiCommandRequest
		wantMethod string
		wantErr bool
	}{
		{name:"power on", req:xiaomiCommandRequest{Action:"set_power",Value:true}, wantMethod:"set_power"},
		{name:"power off", req:xiaomiCommandRequest{Action:"set_power",Value:false}, wantMethod:"set_power"},
		{name:"auto mode", req:xiaomiCommandRequest{Action:"set_mode",Value:"auto"}, wantMethod:"set_mode"},
		{name:"favorite mode", req:xiaomiCommandRequest{Action:"set_mode",Value:"favorite"}, wantMethod:"set_mode"},
		{name:"read properties", req:xiaomiCommandRequest{Action:"read_properties",Properties:[]string{"power","mode"}}, wantMethod:"get_prop"},
		{name:"reject arbitrary method", req:xiaomiCommandRequest{Action:"miIO.reboot"}, wantErr:true},
		{name:"reject non boolean power", req:xiaomiCommandRequest{Action:"set_power",Value:"on"}, wantErr:true},
		{name:"reject unsupported mode", req:xiaomiCommandRequest{Action:"set_mode",Value:"turbo"}, wantErr:true},
		{name:"reject invalid property key", req:xiaomiCommandRequest{Action:"read_properties",Properties:[]string{"power;reboot"}}, wantErr:true},
	}
	for _, tc := range cases {
		t.Run(tc.name,func(t *testing.T) {
			method,_,err:=xiaomiCommandForRequest(tc.req)
			if (err!=nil)!=tc.wantErr { t.Fatalf("error=%v, wantErr=%v",err,tc.wantErr) }
			if err==nil && method!=tc.wantMethod { t.Fatalf("method=%q want %q",method,tc.wantMethod) }
		})
	}
}

func TestXiaomiWriteControlsRequireKnownPurifierModel(t *testing.T) {
	if !isSupportedXiaomiPurifier("zhimi.airpurifier.m1") {
		t.Fatal("known zhimi purifier model should expose the constrained purifier controls")
	}
	for _, model := range []string{"", "rockrobo.vacuum.v1", "unknown.device"} {
		if isSupportedXiaomiPurifier(model) {
			t.Errorf("model %q must not expose purifier-specific write controls", model)
		}
	}
}
