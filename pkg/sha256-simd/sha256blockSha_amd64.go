//+build !noasm,!appengine

package sha256

func blockSha(h *[8]uint32, message []uint8)
