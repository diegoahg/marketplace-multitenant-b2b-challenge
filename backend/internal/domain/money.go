package domain

import (
	"errors"
	"math/big"
	"strings"
)

// Amount is signed minor units. JSON uses a string to preserve all int64 digits.
type Money struct {
	Amount   int64  `json:"amount,string" bson:"amount"`
	Currency string `json:"currency" bson:"currency"`
	Scale    int    `json:"scale" bson:"scale"`
}
type Rounding string

const (
	HalfUp   Rounding = "HALF_UP"
	HalfEven Rounding = "HALF_EVEN"
)

var ErrMoney = errors.New("invalid money, currency mismatch or overflow")

func (m Money) Valid() bool {
	if len(m.Currency) != 3 || m.Scale < 0 || m.Scale > 6 {
		return false
	}
	for _, c := range m.Currency {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
func (m Money) Same(n Money) bool {
	return m.Valid() && n.Valid() && m.Currency == n.Currency && m.Scale == n.Scale
}
func (m Money) WithAmount(n int64) Money { m.Amount = n; return m }
func (m Money) Add(n Money) (Money, error) {
	if !m.Same(n) {
		return Money{}, ErrMoney
	}
	return m.fromBig(new(big.Int).Add(big.NewInt(m.Amount), big.NewInt(n.Amount)))
}
func (m Money) Sub(n Money) (Money, error) {
	if !m.Same(n) {
		return Money{}, ErrMoney
	}
	return m.fromBig(new(big.Int).Sub(big.NewInt(m.Amount), big.NewInt(n.Amount)))
}
func (m Money) Mul(n int64) (Money, error) {
	return m.fromBig(new(big.Int).Mul(big.NewInt(m.Amount), big.NewInt(n)))
}
func (m Money) Compare(n Money) (int, error) {
	if !m.Same(n) {
		return 0, ErrMoney
	}
	return big.NewInt(m.Amount).Cmp(big.NewInt(n.Amount)), nil
}
func (m Money) fromBig(n *big.Int) (Money, error) {
	if !m.Valid() || !n.IsInt64() {
		return Money{}, ErrMoney
	}
	m.Amount = n.Int64()
	return m, nil
}
func (m Money) Ratio(n, d int64, mode Rounding) (Money, error) {
	if d <= 0 || n < 0 || (mode != HalfUp && mode != HalfEven) {
		return Money{}, ErrMoney
	}
	x := new(big.Int).Mul(big.NewInt(m.Amount), big.NewInt(n))
	sign := x.Sign()
	x.Abs(x)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(x, big.NewInt(d), r)
	cmp := new(big.Int).Lsh(r, 1).Cmp(big.NewInt(d))
	if cmp > 0 || (cmp == 0 && (mode == HalfUp || q.Bit(0) == 1)) {
		q.Add(q, big.NewInt(1))
	}
	if sign < 0 {
		q.Neg(q)
	}
	return m.fromBig(q)
}
func (m Money) Percent(bps int64, mode Rounding) (Money, error) { return m.Ratio(bps, 10000, mode) }
func ParseMoney(s, currency string, scale int, mode Rounding) (Money, error) {
	m := Money{Currency: currency, Scale: scale}
	if !m.Valid() || len(s) > 80 || (mode != HalfUp && mode != HalfEven) {
		return Money{}, ErrMoney
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = strings.TrimPrefix(s, "-")
	}
	parts := strings.Split(s, ".")
	n, decimals, err := decimalDigits(parts)
	if err != nil {
		return Money{}, err
	}
	if decimals <= scale {
		n.Mul(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale-decimals)), nil))
	} else {
		n = roundDecimal(n, decimals-scale, mode)
	}
	if negative {
		n.Neg(n)
	}
	return m.fromBig(n)
}
func decimalDigits(parts []string) (*big.Int, int, error) {
	if len(parts) > 2 || parts[0] == "" {
		return nil, 0, ErrMoney
	}
	decimals := 0
	if len(parts) == 2 {
		decimals = len(parts[1])
		if decimals == 0 {
			return nil, 0, ErrMoney
		}
	}
	digits := strings.Join(parts, "")
	for _, c := range digits {
		if c < '0' || c > '9' {
			return nil, 0, ErrMoney
		}
	}
	n, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, 0, ErrMoney
	}
	return n, decimals, nil
}
func roundDecimal(n *big.Int, places int, mode Rounding) *big.Int {
	d := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n, d, r)
	cmp := new(big.Int).Lsh(r, 1).Cmp(d)
	if cmp > 0 || (cmp == 0 && (mode == HalfUp || q.Bit(0) == 1)) {
		q.Add(q, big.NewInt(1))
	}
	return q
}
