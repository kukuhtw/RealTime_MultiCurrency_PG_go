// payment-gateway-poc/tests/integration/load_from_csv_test.go
package integration

import (
	"encoding/csv"
	"os"
	"testing"
)

func TestLoadFromCSV(t *testing.T) {
	f, err := os.Open("../data/dummy_transactions.csv")
	if err != nil { t.Skip("generate csv first via dummygen") }
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil { t.Fatal(err) }
	if len(records) < 2 {
		t.Fatalf("expected >1 rows, got %d", len(records))
	}
}
