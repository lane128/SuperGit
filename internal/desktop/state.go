package desktop

// jobGuard is used only on the main thread. Each read lane has its own sequence;
// switching repositories invalidates all in-flight reads together.
type ticket struct {
	repository, sequence uint64
	lane                 string
}
type jobGuard struct {
	repository uint64
	lanes      map[string]uint64
	writing    bool
}

func (g *jobGuard) Switch() { g.repository++; g.lanes = make(map[string]uint64) }
func (g *jobGuard) Begin(lane string) ticket {
	if g.lanes == nil {
		g.lanes = make(map[string]uint64)
	}
	g.lanes[lane]++
	return ticket{g.repository, g.lanes[lane], lane}
}
func (g *jobGuard) Accept(t ticket) bool {
	return t.repository == g.repository && t.sequence == g.lanes[t.lane]
}
func (g *jobGuard) StartWrite() bool {
	if g.writing {
		return false
	}
	g.writing = true
	return true
}
func (g *jobGuard) EndWrite() { g.writing = false }
