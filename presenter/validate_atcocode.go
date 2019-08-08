package main

import (
	"regexp"
	"strings"
)

func (p *Presenter) validateAtcocode(atcocode string) bool {
	p.Logger.Debugf("validateAtcocode: %s", atcocode)
	atcocode = strings.ToUpper(atcocode)
	matched, _ := regexp.MatchString(`^[A-Z0-9]{8,12}$`, atcocode)
	return matched
}
