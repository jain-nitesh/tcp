package tcp

import "strings"

type Address struct {
	Ip   string
	Port string
}

func NewAddr(address string) *Address {
	addr := new(Address)
	addr.parse(address)
	return addr
}

func (a *Address) parse(address string) {
	params := strings.Split(address, ":")
	if len(params) == 1 {
		a.Ip = params[0]
	} else {
		if params[0] == "" {
			a.Ip = "127.0.0.1"
		} else {
			a.Ip = params[0]
		}
		a.Port = params[1]
	}
}
func (a *Address) GetAddress() string {
	return a.Ip + ":" + a.Port
}
