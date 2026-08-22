package bloom

import (
	"context"
	"math"
)

type Stats struct {
	TaskID    int64   `json:"task_id"`
	Bits      uint64  `json:"bits"`
	Hashes    int     `json:"hashes"`
	SetBits   int64   `json:"set_bits"`
	FillRatio float64 `json:"fill_ratio"`
	EstItems  float64 `json:"est_items"`
	EstFPRate float64 `json:"est_fp_rate"`
}

func (f *Filter) Stats(ctx context.Context, taskID int64) (Stats, error) {
	n, err := f.BitCount(ctx, taskID)
	if err != nil {
		return Stats{}, err
	}
	fill := 0.0
	if f.m > 0 {
		fill = float64(n) / float64(f.m)
	}
	est := 0.0
	if fill > 0 && fill < 1 && f.k > 0 {
		est = -float64(f.m) / float64(f.k) * math.Log(1-fill)
	}
	fp := math.Pow(fill, float64(f.k))
	return Stats{
		TaskID:    taskID,
		Bits:      f.m,
		Hashes:    f.k,
		SetBits:   n,
		FillRatio: fill,
		EstItems:  est,
		EstFPRate: fp,
	}, nil
}

func EstimateFalsePositive(setBits int64, m uint64, k int) float64 {
	if m == 0 || k <= 0 {
		return 1
	}
	fill := float64(setBits) / float64(m)
	if fill < 0 {
		fill = 0
	}
	if fill > 1 {
		fill = 1
	}
	return math.Pow(fill, float64(k))
}
