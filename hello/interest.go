package hello

import (
	"fmt"
	"time"
)

type InterestCalculator struct {
	owed         Cents
	rate         float32
	interest     Cents // Interest accumulated (but not applied) this month
	entries      []Entry
	lastInterest time.Time
}

func (i *InterestCalculator) add(e Entry) {
	if len(i.entries) != 0 {
		i.calcInterest(e.Date)
	}

	switch e.Type {
	case Payment:
		i.owed -= e.Amount
	case Loan:
		i.owed += e.Amount
	case RateChange:
		i.rate = e.Rate
	}
	i.lastInterest = e.Date

	e.Owed = i.owed
	i.entries = append(i.entries, e)
}

func monthLength(month time.Month, year int) int {
	return 30 // FIXME
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
	for !(year == now_year && month == now_month) {
		// Apply interest for remainder of this month
		daysRemainingInThisMonth := monthLength(month, year) - day
		i.interest += i.daysInterest(daysRemainingInThisMonth)

		i.owed += i.interest

		// Next month
		nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, now.Location())
		i.entries = append(i.entries, Entry{
			Date:   nextMonth,
			User:   "",
			Type:   InterestApplied,
			Amount: i.interest,
			Owed:   i.owed,
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
