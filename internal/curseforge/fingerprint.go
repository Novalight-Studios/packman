package curseforge

// Fingerprint computes the CurseForge hash of a file: MurmurHash2 (seed 1)
// over the content stripped of the bytes 9, 10, 13 and 32 (whitespace).
// CurseForge uses it to identify a .jar, see Client.ModIDByFingerprint.
func Fingerprint(data []byte) uint32 {
	b := make([]byte, 0, len(data))
	for _, c := range data {
		if c != 9 && c != 10 && c != 13 && c != 32 {
			b = append(b, c)
		}
	}

	const m uint32 = 0x5bd1e995
	h := uint32(1) ^ uint32(len(b))

	i := 0
	for ; i+4 <= len(b); i += 4 {
		k := uint32(b[i]) | uint32(b[i+1])<<8 | uint32(b[i+2])<<16 | uint32(b[i+3])<<24
		k *= m
		k ^= k >> 24
		k *= m
		h *= m
		h ^= k
	}
	switch len(b) - i {
	case 3:
		h ^= uint32(b[i+2]) << 16
		fallthrough
	case 2:
		h ^= uint32(b[i+1]) << 8
		fallthrough
	case 1:
		h ^= uint32(b[i])
		h *= m
	}
	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}
