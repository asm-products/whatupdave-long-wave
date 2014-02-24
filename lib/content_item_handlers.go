package readraptor

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/codegangsta/martini"
	"github.com/coopernurse/gorp"
	"github.com/cupcake/gokiq"
)

type ContentItemParams struct {
	Key       string           `json:"key"`
	Expected  []string         `json:"expected"`
	Callbacks []CallbackParams `json:"callbacks"`
}

type CallbackParams struct {
	Delay    string   `json:"delay"`
	Expected []string `json:"expected"`
	Url      string   `json:"url"`
}

func GetContentItems(dbmap *gorp.DbMap, params martini.Params) (string, int) {
	var ci ContentItem
	err := dbmap.SelectOne(&ci, "select * from content_items where key = $1", params["content_item_id"])
	ci.AddReadReceipts(dbmap)

	if err != nil {
		panic(err)
	}

	json, err := json.Marshal(ci)
	if err != nil {
		panic(err)
	}

	return string(json), http.StatusOK
}

func PostContentItems(dbmap *gorp.DbMap, client *gokiq.ClientConfig, req *http.Request, account *Account) (string, int) {
	decoder := json.NewDecoder(req.Body)
	var p ContentItemParams
	err := decoder.Decode(&p)
	if err != nil {
		panic(err)
	}

	cid, err := InsertContentItem(dbmap, account.Id, p.Key)
	if err != nil {
		panic(err)
	}

	rids, err := AddContentReaders(dbmap, account.Id, cid, p.Expected)
	for _, callback := range p.Callbacks {
		delay, err := time.ParseDuration(callback.Delay)
		if err != nil {
			panic(err)
		}
		at := time.Now().Add(delay)

		if callback.Expected != nil {
			rids, err = AddContentReaders(dbmap, account.Id, cid, callback.Expected)
            if err != nil {
                panic(err)
            }
		}
		ScheduleCallbacks(client, rids, at, callback.Url)
	}

	ci, err := FindContentItemWithReadReceipts(dbmap, cid)

	json, err := json.Marshal(map[string]interface{}{
		"content_item": ci,
	})
	if err != nil {
		panic(err)
	}
	return string(json), http.StatusCreated
}
