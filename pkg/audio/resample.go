package audio

import (
	"fmt"
)

type ResampleOperator interface {
	// 获取执行参数
	GetRate() (int, int, int)
	// 执行重采样
	Handle(pcm []byte) (npcm []byte, err error)
}

// sample rate resampler.
type goResampler struct {
	channels int // Channels, L or LR
	isr      int // Transform from this sample rate.
	osr      int // Transform to this sample rate.

	// Always cache 16samples.
	lcache []int16 // For channel=0
	rcache []int16 // For channel=1

	// Total outputed samples.
	lws uint64 // For channel=0
	rws uint64 // For channel=1

	// Total consumed samples.
	lcs uint64 // For channel=0
	rcs uint64 // For channel=1
}

// NewGoResampler Create resampler to transform pcm
// from sampleRate to nSampleRate, where pcm contains number of channels
// @remark each sample is 16bits in short int.
func NewGoResampler(channels, sampleRate int, nSampleRate int) (ResampleOperator, error) {
	if channels < 1 || channels > 2 {
		return nil, fmt.Errorf("invalid channels=%v", channels)
	}
	if sampleRate <= 0 {
		return nil, fmt.Errorf("invalid sampleRate=%v", sampleRate)
	}
	if nSampleRate <= 0 {
		return nil, fmt.Errorf("invalid nSampleRate=%v", nSampleRate)
	}

	v := &goResampler{
		channels: channels,
		isr:      sampleRate,
		osr:      nSampleRate,
	}

	return v, nil
}

func (v *goResampler) GetRate() (int, int, int) {
	return v.channels, v.isr, v.osr
}

func (v *goResampler) Handle(pcm []byte) (npcm []byte, err error) {
	if len(pcm) == 0 {
		return nil, fmt.Errorf("empty pcm")
	}
	if (len(pcm) % (2 * v.channels)) != 0 {
		return nil, fmt.Errorf("invalid pcm, should mod(%v)", 2*v.channels)
	}

	if v.isr == v.osr {
		return pcm[:], nil
	}

	// Atleast 4samples when not init.
	if nbSamles := len(pcm) / 2 / v.channels; nbSamles < 4 {
		return nil, fmt.Errorf("invalid pcm, atleast 4samples, actual %vsamples", nbSamles)
	}

	// Convert pcm to int16 values
	ipcmLeft := resampleInitChannel(pcm, v.channels, 0)
	ipcmRight := resampleInitChannel(pcm, v.channels, 1)
	if ipcmRight != nil && len(ipcmLeft) != len(ipcmRight) {
		return nil, fmt.Errorf("invalid pcm, L%v!=%v", len(ipcmLeft), len(ipcmRight))
	}

	// Insert the cache at the beginning.
	if v.lcache != nil {
		ipcmLeft = append(v.lcache, ipcmLeft...)
		v.lcache = nil
	}
	if ipcmRight != nil && v.rcache != nil {
		ipcmRight = append(v.rcache, ipcmRight...)
		v.rcache = nil
	}

	// Resample all channels
	var consumed int
	var opcmLeft []int16
	if opcmLeft, consumed, err = resampleChannel(ipcmLeft, v.isr, v.osr, v.lws, v.lcs); err != nil {
		return nil, err
	}
	v.lws += uint64(len(opcmLeft))
	v.lcs += uint64(consumed)
	if consumed < len(ipcmLeft) {
		v.lcache = ipcmLeft[consumed:]
	}

	var opcmRight []int16
	if ipcmRight != nil {
		if opcmRight, consumed, err = resampleChannel(ipcmRight, v.isr, v.osr, v.rws, v.rcs); err != nil {
			return nil, err
		}
		v.rws += uint64(len(opcmRight))
		v.rcs += uint64(consumed)
		if consumed < len(ipcmRight) {
			v.rcache = ipcmRight[consumed:]
		}
	}

	// Convert int16 samples to bytes.
	npcm = resampleMerge(opcmLeft, opcmRight)

	return
}

// merge left and right(can be nil).
func resampleMerge(left, right []int16) (npcm []byte) {
	npcm = []byte{}
	for i := 0; i < len(left); i++ {
		v := left[i]
		npcm = append(npcm, byte(v))
		npcm = append(npcm, byte(v>>8))

		if right != nil {
			v = right[i]
			npcm = append(npcm, byte(v))
			npcm = append(npcm, byte(v>>8))
		}
	}
	return
}

// x is the position of output pcm
func resampleChannel(ipcm []int16, isr, osr int, written, org uint64) (opcm []int16, consumed int, err error) {
	if len(ipcm) <= 16 {
		return nil, 0, nil
	}

	// The samples we can use to resample
	available := len(ipcm) - 16
	// The resample step between new samples
	step := float64(isr) / float64(osr)
	// The first position to sample
	x0 := step * float64(written)

	// The position for the last sample.
	last := org + uint64(available)

	// Resample each position from x0
	for x := x0; x < float64(last); x += step {
		// Generate xi,yi,xo,yo
		xi0 := float64(uint64(x))
		xi := []float64{xi0, xi0 + 1, xi0 + 2, xi0 + 3}
		yi0 := int(uint64(xi0) - org)
		yi := []float64{float64(ipcm[yi0]), float64(ipcm[yi0+1]), float64(ipcm[yi0+2]), float64(ipcm[yi0+3])}
		xo := []float64{x}
		yo := []float64{0.0}
		if err = spline(xi, yi, xo, yo); err != nil {
			return
		}

		// convert yo
		opcm = append(opcm, int16(yo[0]))
		consumed = int(uint64(x)-org) + 1
	}

	return
}

func resampleInitChannel(pcm []byte, channels, channel int) (ipcm []int16) {
	if channel >= channels {
		return
	}

	ipcm = []int16{}
	for i := 2 * channel; i < len(pcm); i += 2 * channels {
		// 16bits le sample
		v := (int16(pcm[i])) | (int16(pcm[i+1]) << 8)
		ipcm = append(ipcm, v)
	}

	return
}

func spline(xi, yi, xo, yo []float64) (err error) {
	if len(xi) != 4 {
		return fmt.Errorf("invalid xi")
	}
	if len(yi) != 4 {
		return fmt.Errorf("invalid yi")
	}
	if len(xo) == 0 {
		return fmt.Errorf("invalid xo")
	}
	if len(yo) != len(xo) {
		return fmt.Errorf("invalid yo")
	}

	x0, x1, x2, x3 := xi[0], xi[1], xi[2], xi[3]
	y0, y1, y2, y3 := yi[0], yi[1], yi[2], yi[3]
	h0, h1, h2, _, u1, l2, _ := spline_lu(xi)
	c1, c2 := spline_c1(yi, h0, h1), spline_c2(yi, h1, h2)
	m1, m2 := spline_m1(c1, c2, u1, l2), spline_m2(c1, c2, u1, l2) // m0=m3=0

	for k, v := range xo {
		if v <= x1 {
			yo[k] = spline_z0(m1, h0, x0, x1, y0, y1, v)
		} else if v <= x2 {
			yo[k] = spline_z1(m1, m2, h1, x1, x2, y1, y2, v)
		} else {
			yo[k] = spline_z2(m2, h2, x2, x3, y2, y3, v)
		}
	}

	return
}

func spline_z0(m1, h0, x0, x1, y0, y1, x float64) float64 {
	v0 := 0.0
	v1 := (x - x0) * (x - x0) * (x - x0) * m1 / (6 * h0)
	v2 := -1.0 * y0 * (x - x1) / h0
	v3 := (y1 - h0*h0*m1/6) * (x - x0) / h0
	return v0 + v1 + v2 + v3
}

func spline_z1(m1, m2, h1, x1, x2, y1, y2, x float64) float64 {
	v0 := -1.0 * (x - x2) * (x - x2) * (x - x2) * m1 / (6 * h1)
	v1 := (x - x1) * (x - x1) * (x - x1) * m2 / (6 * h1)
	v2 := -1.0 * (y1 - h1*h1*m1/6) * (x - x2) / h1
	v3 := (y2 - h1*h1*m2/6) * (x - x1) / h1
	return v0 + v1 + v2 + v3
}

func spline_z2(m2, h2, x2, x3, y2, y3, x float64) float64 {
	v0 := -1.0 * (x - x3) * (x - x3) * (x - x3) * m2 / (6 * h2)
	v1 := 0.0
	v2 := -1.0 * (y2 - h2*h2*m2/6) * (x - x3) / h2
	v3 := y3 * (x - x2) / h2
	return v0 + v1 + v2 + v3
}

// Calculate M1 form matrix.
func spline_m1(c1, c2, u1, l2 float64) float64 {
	return (c1/u1 - c2/2) / (2/u1 - l2/2)
}

// Calculate M2 form matrix.
func spline_m2(c1, c2, u1, l2 float64) float64 {
	return (c1/2 - c2/l2) / (u1/2 - 2/l2)
}

func spline_c1(yi []float64, h0, h1 float64) float64 {
	y0, y1, y2, _ := yi[0], yi[1], yi[2], yi[3]
	return 6.0 / (h0 + h1) * ((y2-y1)/h1 - (y1-y0)/h0)
}

func spline_c2(yi []float64, h1, h2 float64) float64 {
	_, y1, y2, y3 := yi[0], yi[1], yi[2], yi[3]
	return 6.0 / (h1 + h2) * ((y3-y2)/h2 - (y2-y1)/h1)
}

func spline_lu(xi []float64) (h0, h1, h2, l1, u1, l2, u2 float64) {
	x0, x1, x2, x3 := xi[0], xi[1], xi[2], xi[3]

	h0, h1, h2 = x1-x0, x2-x1, x3-x2

	l1 = h0 / (h1 + h0) // lambada1
	u1 = h1 / (h1 + h0)

	l2 = h1 / (h2 + h1) // lambada2
	u2 = h2 / (h2 + h1)

	return
}
