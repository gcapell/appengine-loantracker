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

type Greeting struct {
	Author  string
	Content string
	Date    time.Time
}

type Payment struct {
	Date	time.Time
	User	string
	Amount Cents
}

type RateChange struct {
	Date 	time.Time
	User	string
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
	payments, err := getPayments(c)
	if err != nil {
		return err
	}
	return  paymentTemplate.Execute(w, payments)
}

func getPayments(c appengine.Context) ([]Payment, error) {
	q := datastore.NewQuery("Payment").Order("Date")
	
	var payments []Payment
	for t := q.Run(c); ; {
		var p Payment
		key, err := t.Next(&p)
		log.Println("key", key)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, nil
}


func changeRate(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	rate, err := strconv.ParseFloat(r.FormValue("rate"), 32)
	if err != nil {
		return err
	}

	date, err := time.Parse("2 Jan 2006", r.FormValue("date"))
	if err != nil {
		return err
	}
	log.Println("rate", rate, "date", date)
	userName := ""
	if u := user.Current(c); u != nil {
		userName = u.String()
	}
	p := RateChange{date, userName, float32(rate)}
	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "RateChange", nil), &p); err != nil {
		return err
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

func addPayment(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	amount, err := ParseCents(r.FormValue("amount"))
	if err != nil {
		return err
	}

	date, err := time.Parse("2 Jan 2006", r.FormValue("date"))
	if err != nil {
		return err
	}
	log.Println("amount", amount, "date", date)
	userName := ""
	if u := user.Current(c); u != nil {
		userName = u.String()
	}
	p := Payment{date, userName, amount}
	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Payment", nil), &p); err != nil {
		return err
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
