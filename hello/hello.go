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

func (e *Entry) Deletable() bool {
	return e.Key != nil  && time.Now().Sub(e.Date).Hours() < (14*24)
}

func (e *Entry) Description() string {
	switch e.Type {
		case Payment: return "Payment"
		case Loan: return "Loan"
		case RateChange: return "Rate Change"
		case InterestApplied: return "Interest Applied"
	}
	log.Print("Unknown type:", e.Type)
	return ""
}

func (e *Entry) ValueString() string {

	switch e.Type {
		case Payment,Loan, InterestApplied: return e.Amount.String()
		case RateChange: return fmt.Sprintf("%.1f%%", e.Rate )
	}
	log.Print("Unknown type:", e.Type)
	return ""	
}
type Entry struct {
	Date	time.Time
	User	string
	Type	EntryType
	Amount Cents
	Rate	float32	// 3.5 -> 3.5%
	Owed	Cents
	Key     *datastore.Key
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

type InterestCalculator struct {
	owed Cents
	rate	float32
	interest Cents	// Interest accumulated (but not applied) this month
	entries	[]Entry
	lastInterest	time.Time
}

func (i *InterestCalculator) add(e Entry) {
	if len(i.entries)!=0 {
		i.calcInterest(e.Date)
	}
	
	switch(e.Type) {
	case Payment:
		i.owed -= e.Amount
	case Loan:
		i.owed += e.Amount
	case RateChange:
		i.rate = e.Rate
	}
	i.lastInterest = e.Date

	log.Printf("After %q, owe %s", e, i.owed)
	e.Owed = i.owed
	i.entries = append(i.entries, e)
}

func monthLength(month time.Month, year int) int {
	return 30	// FIXME
}

// Calculate daily interest, add monthly interest payments
func (i *InterestCalculator) calcInterest(now time.Time) {
	// Add monthly interest charges for each month-change
	// between lastInterest and now
	
	if now.Before(i.lastInterest) {
		panic(fmt.Sprintf("lastInterest:%q, now:%q", i.lastInterest, now))
	}

	// Insert interest charges at each 1st of month
	// between lastInterest and now

	year, month, day := i.lastInterest.Date()
	now_year, now_month, now_day := now.Date()
	for !(year==now_year && month == now_month) {
		// Apply interest for remainder of this month
		daysRemainingInThisMonth := monthLength(month, year) - day
		i.interest += i.daysInterest(daysRemainingInThisMonth)

		i.owed += i.interest

		// Next month
		nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, now.Location())
		i.entries = append(i.entries, Entry {
			Date: nextMonth,
			User:"", 
			Type: InterestApplied, 
			Amount: i.interest,
			Owed: i.owed,
		})
		
		year, month, day = nextMonth.Date()
		i.interest = 0
	}

	// Calculate interest between start of month and now
	i.interest = i.daysInterest(now_day)
	i.lastInterest = now
}

func (i *InterestCalculator) daysInterest(days int) Cents {
	return Cents(i.rate / 100 * float32(i.owed) * float32(days) / 365)
}


func getEntries(c appengine.Context) ([]Entry, error) {
	q := datastore.NewQuery("Entry").Order("Date")
	
	i := new( InterestCalculator)

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
		e.Key = key
		i.add(e)
	}
	return i.entries, nil
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

	date, err := time.Parse("2006-01-02", r.FormValue("date"))
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
