//+build !noasm,!appengine

package sha256

func blockAvx2Go(dig *digest, p []byte) {}
func blockAvxGo(dig *digest, p []byte)  {}
func blockSsseGo(dig *digest, p []byte) {}
func blockShaGo(dig *digest, p []byte)  {}

func blockArm(h []uint32, message []uint8)

func blockArmGo(dig *digest, p []byte) {
	h := []uint32{dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7]}

	blockArm(h[:], p[:])

	dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7] = h[0], h[1], h[2], h[3], h[4],
		h[5], h[6], h[7]
}
