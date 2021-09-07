// +build !go1.8
// +build amd64 !gccgo !appengine !nacl

package highwayhash

import "golang.org/x/sys/cpu"

var (
	useSSE4 = cpu.X86.HasSSE41
	useAVX2 = false
	useNEON = false
	useVMX  = false
)

func initializeSSE4(state *[16]uint64, key []byte)

func updateSSE4(state *[16]uint64, msg []byte)

func finalizeSSE4(out []byte, state *[16]uint64)

func initialize(state *[16]uint64, key []byte) {
	if useSSE4 {
		initializeSSE4(state, key)
	} else {
		initializeGeneric(state, key)
	}
}

func update(state *[16]uint64, msg []byte) {
	if useSSE4 {
		updateSSE4(state, msg)
	} else {
		updateGeneric(state, msg)
	}
}

func finalize(out []byte, state *[16]uint64) {
	if useSSE4 {
		finalizeSSE4(out, state)
	} else {
		finalizeGeneric(out, state)
	}
}
