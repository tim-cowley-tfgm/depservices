package main

func (p *Presenter) validateTop(top int64) bool {
	p.Logger.Debugf("validateTop: %d", top)
	return top > 0
}
