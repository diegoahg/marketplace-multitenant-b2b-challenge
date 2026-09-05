package domain

import (
	"encoding/json"
	"math"
	"testing"
)

func TestGT09Rounding(t *testing.T) {
	for _, tc := range []struct {
		s    string
		mode Rounding
		want int64
	}{{"0.005", HalfUp, 1}, {"1.005", HalfUp, 101}, {"10.015", HalfUp, 1002}, {"1.005", HalfEven, 100}, {"10.015", HalfEven, 1002}, {"-1.005", HalfUp, -101}} {
		t.Run(tc.s+string(tc.mode), func(t *testing.T) {
			m, e := ParseMoney(tc.s, "PEN", 2, tc.mode)
			if e != nil || m.Amount != tc.want {
				t.Fatalf("got %v %v, want %d", m, e, tc.want)
			}
		})
	}
}
func TestMoneySafety(t *testing.T) {
	if _, err := ParseMoney("1.00", "PEN", 2, "UNKNOWN"); err == nil { t.Fatal("invalid rounding accepted") }
	if (Money{Amount: 1, Currency: "pen", Scale: 2}).Valid() { t.Fatal("invalid currency accepted") }
	m := Money{math.MaxInt64, "PEN", 2}
	if _, e := m.Add(m); e == nil {
		t.Fatal("overflow accepted")
	}
	if _, e := m.Sub(Money{1, "CLP", 0}); e == nil {
		t.Fatal("mixed currency accepted")
	}
	if _, e := m.Mul(2); e == nil {
		t.Fatal("overflow accepted")
	}
	for _, s := range []string{"NaN", "1e3", "1/2", "1.", "--2"} {
		if _, e := ParseMoney(s, "PEN", 2, HalfUp); e == nil {
			t.Fatal("invalid accepted", s)
		}
	}
	data, e := json.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	var roundtrip Money
	if e = json.Unmarshal(data, &roundtrip); e != nil || roundtrip != m {
		t.Fatal("precision lost", string(data), e)
	}
}
