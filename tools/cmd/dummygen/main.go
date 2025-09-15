// payment-gateway-poc/tools/cmd/dummygen/main.go
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	n := flag.Int("n", 100, "jumlah baris data (tanpa header)")
	out := flag.String("out", "tests/data/dummy_transactions.csv", "path output CSV")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	if err := os.MkdirAll("tests/data", 0o755); err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	_ = w.Write([]string{"id", "currency", "amount", "source_account", "destination_account"})
	for i := 0; i < *n; i++ {
		row := []string{
			fmt.Sprintf("PAY-%06d", i+1),
			[]string{"USD", "IDR", "SGD", "EUR"}[rand.Intn(4)],
			fmt.Sprintf("%.2f", 10+rand.Float64()*1000),
			fmt.Sprintf("ACC_SRC_%04d", rand.Intn(10000)),
			fmt.Sprintf("ACC_DST_%04d", rand.Intn(10000)),
		}
		if err := w.Write(row); err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("generated %s (%d rows + header)", *out, *n)
}
