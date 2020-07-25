package ec

const escapeByte byte = 7

func escape(pkg []byte) []byte {
	r := make([]byte, 0, len(pkg))
	for _, e := range pkg {
		switch e {
		case PkgStart:
			r = append(r, escapeByte)
			r = append(r, 8)
		case PkgEnd:
			r = append(r, escapeByte)
			r = append(r, 9)
		case ControlPkgStart:
			r = append(r, escapeByte)
			r = append(r, 10)
		case ControlPkgEnd:
			r = append(r, escapeByte)
			r = append(r, 11)
		case Ignore:
			r = append(r, escapeByte)
			r = append(r, 12)
		case escapeByte:
			r = append(r, escapeByte)
			r = append(r, escapeByte)
		default:
			r = append(r, e)
		}
	}
	return r
}

func unescape(e []byte) []byte {
	r := make([]byte, 0, len(e))
	for i := 0; i < len(e); i++ {
		switch e[i] {
		case escapeByte:
			i++
			switch e[i] {
			case 8:
				r = append(r, PkgStart)
			case 9:
				r = append(r, PkgEnd)
			case 10:
				r = append(r, ControlPkgStart)
			case 11:
				r = append(r, ControlPkgEnd)
			case 12:
				r = append(r, Ignore)
			case escapeByte:
				r = append(r, escapeByte)
			}
		default:
			r = append(r, e[i])
		}
	}
	return r
}
