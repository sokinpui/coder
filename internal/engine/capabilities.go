package engine

type Capability uint32

const (
	CapContextFiles Capability = 1 << iota
	CapToolToggle
	CapITF
	CapModelSwitch
	CapBranch
	CapRegenerate
	CapTerminal
	CapDocumentContext
	CapToolLoop
)

func (c Capability) Has(target Capability) bool {
	return (c & target) != 0
}
