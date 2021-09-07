// +build go1.8
// +build amd64 !gccgo !appengine !nacl

package highwayhash

import "golang.org/x/sys/cpu"

var (
	useSSE4 = cpu.X86.HasSSE41
	useAVX2 = cpu.X86.HasAVX2
	useNEON = false
	useVMX  = false
)

func initializeSSE4(state *[16]uint64, key []byte)

func initializeAVX2(state *[16]uint64, key []byte)

func updateSSE4(state *[16]uint64, msg []byte)

func updateAVX2(state *[16]uint64, msg []byte)

func finalizeSSE4(out []byte, state *[16]uint64)

func finalizeAVX2(out []byte, state *[16]uint64)

func initialize(state *[16]uint64, key []byte) {
	switch {
	case useAVX2:
		initializeAVX2(state, key)
	case useSSE4:
		initializeSSE4(state, key)
	default:
		initializeGeneric(state, key)
	}
}

func update(state *[16]uint64, msg []byte) {
	switch {
	case useAVX2:
		updateAVX2(state, msg)
	case useSSE4:
		updateSSE4(state, msg)
	default:
		updateGeneric(state, msg)
	}
}

func finalize(out []byte, state *[16]uint64) {
	switch {
	case useAVX2:
		finalizeAVX2(out, state)
	case useSSE4:
		finalizeSSE4(out, state)
	default:
		finalizeGeneric(out, state)
	}
}
