//+build !noasm,!appengine

package sha256

func blockAvx(h []uint32, message []uint8, reserved0, reserved1, reserved2, reserved3 uint64)
