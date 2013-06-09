package hello

import (
	"appengine/datastore"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"
)

type Cents int

type EntryType int

const (
	Payment EntryType = iota
	Loan
	RateChange
	InterestApplied
)

func (e *Entry) Deletable() bool {
	return e.Key != nil && time.Now().Sub(e.Date).Hours() < (14*24)
}

func (e *Entry) Description() string {
	switch e.Type {
	case Payment:
		return "Payment"
	case Loan:
		return "Loan"
	case RateChange:
		return "Rate Change"
	case InterestApplied:
		return "Interest Applied"
	}
	log.Print("Unknown type:", e.Type)
	return ""
}

func (e *Entry) ValueString() string {
	switch e.Type {
	case Payment, Loan, InterestApplied:
		return e.Amount.String()
	case RateChange:
		return fmt.Sprintf("%.1f%%", e.Rate)
	}
	log.Print("Unknown type:", e.Type)
	return ""
}

type Entry struct {
	Date   time.Time
	User   string
	Type   EntryType
	Amount Cents
	Rate   float32 // 3.5 -> 3.5%
	Owed   Cents
	Key    *datastore.Key
}

func (c Cents) String() string {
	dollars, cents := c/100, c%100
	return fmt.Sprintf("$%d.%02d", dollars, cents)
}

var amountRE = regexp.MustCompile(`(\d+)(.\d\d)?`)

func ParseCents(s string) (Cents, error) {
	matches := amountRE.FindStringSubmatch(s)
	log.Println("matches", matches)
	if matches == nil || len(matches) < 2 {
		return 0, fmt.Errorf("could not parse %q into dollar amount", s)
	}
	dollars, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("Bad dollar amount:%q", matches[1])
	}
	cents := 0
	if len(matches[2]) > 1 {
		cents, err = strconv.Atoi(matches[2][1:])
		if err != nil {
			return 0, fmt.Errorf("ParseCents: Bad cents amount:%q in %q", matches[2], s)
		}
	}

	return Cents(dollars*100 + cents), nil
}
