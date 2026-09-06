// Run inside the backend module; see README. Outputs real pricing snapshots for UI tests.
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
    d "marketplace/internal/domain"
    "marketplace/internal/pricing"
    "marketplace/internal/testkit"
)

func main() {
    cases := []struct {
        name string
        items []d.CartItem
        rules []d.Promotion
        credit int64
    }{
        {"GT01", []d.CartItem{{SKU: "A", Quantity: 12}}, []d.Promotion{testkit.Scale()}, 100000},
        {"GT02", []d.CartItem{{SKU: "A", Quantity: 8}, {SKU: "B", Quantity: 1}}, []d.Promotion{testkit.Combo(), testkit.Scale()}, 100000},
        {"GT03", []d.CartItem{{SKU: "A", Quantity: 6}}, []d.Promotion{testkit.Gift()}, 100000},
        {"GT04", []d.CartItem{{SKU: "A", Quantity: 2}, {SKU: "B", Quantity: 1}}, []d.Promotion{testkit.Combo()}, 100000},
        {"GT05", []d.CartItem{{SKU: "A", Quantity: 2}}, nil, 235},
    }
    for i, tc := range cases {
        q, err := pricing.Calculate(testkit.Cart(tc.items...), testkit.Config(), testkit.Products(), tc.rules, testkit.Config().Money(tc.credit), testkit.Now())
        if err != nil { panic(err) }
        q.QuoteID = fmt.Sprintf("11111111-1111-4111-8111-%012d", i+1)
        q.CreatedAt = testkit.Now()
        q.ExpiresAt = q.CreatedAt.Add(15*time.Minute)
        raw, err := json.MarshalIndent(q, "", "  ")
        if err != nil { panic(err) }
        if err := os.WriteFile(filepath.Join(os.Args[1], "quote_"+tc.name+".json"), raw, 0644); err != nil { panic(err) }
    }
}
