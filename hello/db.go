package hello

import (
		"appengine"
		"log"
		"appengine/datastore"
)
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

