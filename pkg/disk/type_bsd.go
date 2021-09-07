// +build darwin freebsd dragonfly openbsd solaris

package disk

func getFSType(fstype []int8) string {
	b := make([]byte, len(fstype))
	for i, v := range fstype {
		b[i] = byte(v)
	}
	return string(b)
}
