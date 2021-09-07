//+build !noasm

package highwayhash

var (
	useSSE4 = false
	useAVX2 = false
	useNEON = false
	useVMX  = true
)

func updatePpc64Le(state *[16]uint64, msg []byte)

func initialize(state *[16]uint64, key []byte) {
	initializeGeneric(state, key)
}

func update(state *[16]uint64, msg []byte) {
	if useVMX {
		updatePpc64Le(state, msg)
	} else {
		updateGeneric(state, msg)
	}
}

func finalize(out []byte, state *[16]uint64) {
	finalizeGeneric(out, state)
}
