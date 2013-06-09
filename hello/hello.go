package hello

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func init() {
	http.HandleFunc("/favicon.ico", favicon)
	http.Handle("/", appHandler(root))
	http.Handle("/addPayment", appHandler(addPayment))
	http.Handle("/rate", appHandler(rateForm))
	http.Handle("/changeRate", appHandler(changeRate))
	http.Handle("/delete", appHandler(deleter))
}

type appHandler func(http.ResponseWriter, *http.Request, appengine.Context, *user.User) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}
	if err := fn(w, r, c, u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func favicon(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

func rateForm(w http.ResponseWriter, r *http.Request, c appengine.Context, u *user.User) error {
	return rateTemplate.Execute(w, nil)
}

func root(w http.ResponseWriter, r *http.Request, c appengine.Context, u *user.User) error {
	entries, err := getEntries(c)
	if err != nil {
		return err
	}
	return paymentTemplate.Execute(w, entries)
}

func changeRate(w http.ResponseWriter, r *http.Request, c appengine.Context, u *user.User) error {
	return addEntry(w, r, c, u, RateChange)
}

func addPayment(w http.ResponseWriter, r *http.Request, c appengine.Context, u *user.User) error {
	return addEntry(w, r, c, u, Payment)
}

func addEntry(w http.ResponseWriter, r *http.Request, c appengine.Context, u *user.User, t EntryType) error {
	date, err := time.Parse("2006-01-02", r.FormValue("date"))
	if err != nil {
		return err
	}

	e := Entry{Date: date, User: u.String(), Type: t}
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

func deleter(w http.ResponseWriter, r *http.Request, c appengine.Context, u *user.User) error {
	keyID := r.FormValue("KeyID")
	log.Print("deleter", keyID)
	if keyID == "" {
		return fmt.Errorf("attempt to delete with null key")
	}
	key, err := datastore.DecodeKey(keyID)
	log.Print("err", err, "key", key)
	if err != nil {
		return err
	}
	var e Entry
	if err = datastore.Get(c, key, &e); err != nil {
		return err
	}
	e.Deleted = true
	e.DeleteUser = u.String()
	if _, err = datastore.Put(c, key, &e); err != nil {
		return err
	}
	log.Print("redirecting")
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
