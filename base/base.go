package base

import (
	"github.com/er1cw00/claw.go/base/logger"
)

var DumpSettings = false

func Start(path string) error {
	err := parseSettings(path)
	if err != nil {
		return err
	}
	s := GetSettings()
	if err = logger.New(s.Log.Level, s.Log.Path, "goclaw"); err != nil {
		return err
	}
	if DumpSettings {
		logger.Infof("setting:\n\r%v", s)
	}
	return nil
}

func Stop() {

}
