package syssetup

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

var (
	callIPTablesFile = "/proc/sys/net/bridge/bridge-nf-call-iptables"
)

func Configure() error {
	if err := ioutil.WriteFile(callIPTablesFile, []byte("1"), 0640); err != nil {
		logrus.Warnf("failed to write value 1 at %s: %v", callIPTablesFile, err)
	}
	return nil
}
