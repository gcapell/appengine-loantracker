package hello

import (
	"appengine"
	"appengine/datastore"
	"appengine/user"
	"html/template"
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
	if len(matches)==3 {
		cents, err = strconv.Atoi(matches[2][1:])
		if err != nil {
			return 0, fmt.Errorf("ParseCents: Bad cents amount:%q in %q", matches[2], s)
		}
	}

	return Cents(dollars * 100 + cents), nil
}


var paymentTemplate = template.Must(template.New("book").Parse(paymentTemplateHTML))

const paymentTemplateHTML = `
<html>
  <body>
   <table>
    {{range .}}
	<tr>
	<td>{{.Amount}}</td>
	<td>{{.Date.Format "15:04 2 Jan 2006"}}</td>
	</tr>
	{{end}}
    <form action="/addPayment" method="post">
	  Amount: <input type="text" name="amount"><br>
	  Date: <input type="text" name="date"><br>
      <div><input type="submit" value="Add amount"></div>
    </form>
  </body>
</html>
`

func init() {
	http.HandleFunc("/favicon.ico", favicon)
	http.Handle("/", appHandler(root))
	http.Handle("/addPayment", appHandler(addPayment))
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
	q := datastore.NewQuery("Payment").Order("-Date").Limit(10)
	payments := make([]Payment, 0, 10)
	if _, err := q.GetAll(c, &payments); err != nil {
		return err
	}
	return  paymentTemplate.Execute(w, payments)
}

func addPayment(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	amount, err := ParseCents(r.FormValue("amount"))
	if err != nil {
		return err
	}
	dateString := r.FormValue("date")
	log.Println("amount", amount, "date", dateString)
	userName := ""
	if u := user.Current(c); u != nil {
		userName = u.String()
	}
	p := Payment{time.Now(), userName, amount}
	if _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "Payment", nil), &p); err != nil {
		return err
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
