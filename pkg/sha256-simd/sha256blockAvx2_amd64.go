//+build !noasm,!appengine

package sha256

func blockAvx2(h []uint32, message []uint8)
