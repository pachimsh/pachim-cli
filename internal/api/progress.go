package api

import "io"

type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	onProgress func(int)
	lastPct    int
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	if n > 0 {
		p.read += int64(n)
		if p.total > 0 && p.onProgress != nil {
			pct := int(p.read * 100 / p.total)
			if pct > 100 {
				pct = 100
			}
			if pct != p.lastPct {
				p.lastPct = pct
				p.onProgress(pct)
			}
		}
	}

	return n, err
}
