//+build !noasm,!appengine

package sha256

func blockArmGo(dig *digest, p []byte) {}

func blockAvxGo(dig *digest, p []byte) {
	h := []uint32{dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7]}

	blockAvx(h[:], p[:], 0, 0, 0, 0)

	dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7] = h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]
}

func blockAvx2Go(dig *digest, p []byte) {
	h := []uint32{dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7]}

	blockAvx2(h[:], p[:])

	dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7] = h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]
}

func blockSsseGo(dig *digest, p []byte) {
	h := []uint32{dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7]}

	blockSsse(h[:], p[:], 0, 0, 0, 0)

	dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4], dig.h[5], dig.h[6], dig.h[7] = h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]
}

func blockShaGo(dig *digest, p []byte) {
	blockSha(&dig.h, p)
}
