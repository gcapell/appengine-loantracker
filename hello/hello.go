package hello

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"net/http"
	"time"
	"log"
	"fmt"
	"regexp"
	"strconv"
)

type Cents int

type EntryType int

const (
	Payment EntryType = iota
	Loan
	RateChange
	InterestApplied
)

func(e EntryType) IsPayment () bool {
	return e == Payment
}

func (e EntryType) IsRateChange() bool {
	return e == RateChange
}

func (e EntryType) IsLoan() bool {
	return e == Loan
}

type Entry struct {
	Date	time.Time
	User	string
	Type	EntryType
	Amount Cents
	Rate	float32	// 3.5 -> 3.5%
}

func (c Cents) String() string {
	dollars, cents := c/100, c%100
	return fmt.Sprintf("$%d.%02d", dollars, cents)
}

var amountRE = regexp.MustCompile(`(\d+)(.\d\d)?`)

func ParseCents(s string) (Cents, error) {
	matches := amountRE.FindStringSubmatch(s)
	log.Println("matches", matches)
	if matches == nil || len(matches)<2 {
		return 0, fmt.Errorf("could not parse %q into dollar amount", s)
	}
	dollars, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("Bad dollar amount:%q", matches[1])
	}
	cents := 0
	if len(matches[2])>1 {
		cents, err = strconv.Atoi(matches[2][1:])
		if err != nil {
			return 0, fmt.Errorf("ParseCents: Bad cents amount:%q in %q", matches[2], s)
		}
	}

	return Cents(dollars * 100 + cents), nil
}


func init() {
	http.HandleFunc("/favicon.ico", favicon)
	http.Handle("/", appHandler(root))
	http.Handle("/addPayment", appHandler(addPayment))
	http.Handle("/rate", appHandler(rateForm))
	http.Handle("/changeRate", appHandler(changeRate))
}

type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w,r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func favicon(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

func rateForm(w http.ResponseWriter, r *http.Request) error {
	return rateTemplate.Execute(w, nil)
}

func root(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			return err
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return nil
	}
	entries, err := getEntries(c)
	if err != nil {
		return err
	}
	return  paymentTemplate.Execute(w, entries)
}

func getEntries(c appengine.Context) ([]Entry, error) {
	q := datastore.NewQuery("Entry").Order("Date")
	
	var entries []Entry
	for t := q.Run(c); ; {
		var e Entry
		key, err := t.Next(&e)
		log.Println("key", key)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func changeRate(w http.ResponseWriter, r *http.Request) error {
	return addEntry(w,r, RateChange)
}

func addPayment(w http.ResponseWriter, r *http.Request) error {
	return addEntry(w,r, Payment)
}

func addEntry(w http.ResponseWriter, r *http.Request, t EntryType) error{
	c := appengine.NewContext(r)
	userName := ""
	if u := user.Current(c); u != nil {
		userName = u.String()
	}

	date, err := time.Parse("2 Jan 2006", r.FormValue("date"))
	if err != nil {
		return err
	}

	e := Entry{Date:date, User:userName, Type:t}
	switch t {
	case Payment:
		amount, err := ParseCents(r.FormValue("amount"))
		if err != nil {
			return err
		}
		e.Amount = amount
		if len(r.FormValue("IsLoan")) != 0 {
			e.Type = Loan
		}
	case RateChange:
		rate, err := strconv.ParseFloat(r.FormValue("rate"), 32)
		if err != nil {
			return err
		}
		e.Rate = float32(rate)
	}

	log.Println("entry", e)
	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Entry", nil), &e); err != nil {
		return err
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
