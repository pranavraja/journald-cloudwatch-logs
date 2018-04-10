package main

type Priority int

var (
	EMERGENCY Priority = 0
	ALERT     Priority = 1
	CRITICAL  Priority = 2
	ERROR     Priority = 3
	WARNING   Priority = 4
	NOTICE    Priority = 5
	INFO      Priority = 6
	DEBUG     Priority = 7
)

var PriorityJSON = map[Priority][]byte{
	EMERGENCY: []byte("\"EMERG\""),
	ALERT:     []byte("\"ALERT\""),
	CRITICAL:  []byte("\"CRITICAL\""),
	ERROR:     []byte("\"ERROR\""),
	WARNING:   []byte("\"WARNING\""),
	NOTICE:    []byte("\"NOTICE\""),
	INFO:      []byte("\"INFO\""),
	DEBUG:     []byte("\"DEBUG\""),
}

type Record struct {
	InstanceId string   `json:"instanceId,omitempty"`
	TimeNsec   int64    `json:"-"`
	Command    string   `json:"cmdName,omitempty" journald:"_COMM"`
	Priority   Priority `json:"priority" journald:"PRIORITY"`
	Message    string   `json:"message" journald:"MESSAGE"`
	MessageId  string   `json:"messageId,omitempty" journald:"MESSAGE_ID"`
}

type RecordSyslog struct {
	Facility   int    `json:"facility,omitempty" journald:"SYSLOG_FACILITY"`
	Identifier string `json:"ident,omitempty" journald:"SYSLOG_IDENTIFIER"`
	PID        int    `json:"pid,omitempty" journald:"SYSLOG_PID"`
}

type RecordKernel struct {
	Device    string `json:"device,omitempty" journald:"_KERNEL_DEVICE"`
	Subsystem string `json:"subsystem,omitempty" journald:"_KERNEL_SUBSYSTEM"`
	SysName   string `json:"sysName,omitempty" journald:"_UDEV_SYSNAME"`
	DevNode   string `json:"devNode,omitempty" journald:"_UDEV_DEVNODE"`
}

func (p Priority) MarshalJSON() ([]byte, error) {
	return PriorityJSON[p], nil
}
