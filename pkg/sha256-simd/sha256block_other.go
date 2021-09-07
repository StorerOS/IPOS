//+build appengine noasm !amd64,!arm64

package sha256

func blockAvx2Go(dig *digest, p []byte) {}
func blockAvxGo(dig *digest, p []byte)  {}
func blockSsseGo(dig *digest, p []byte) {}
func blockShaGo(dig *digest, p []byte)  {}
func blockArmGo(dig *digest, p []byte)  {}
