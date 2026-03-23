package templates

import (
	"encoding/json"
	"fmt"
)

type lineItemJSON struct {
	Desc  string  `json:"desc"`
	Qty   float64 `json:"qty"`
	Price float64 `json:"price"`
}

type invoiceFormData struct {
	Items   []lineItemJSON `json:"items"`
	TaxRate float64        `json:"taxRate"`
}

// lineItemsJSON returns the JSON initialization data for the Alpine.js invoice form.
func lineItemsJSON(inv *InvoiceView, taxRate float64) string {
	data := invoiceFormData{TaxRate: taxRate}

	if inv != nil && len(inv.LineItems) > 0 {
		for _, li := range inv.LineItems {
			data.Items = append(data.Items, lineItemJSON{
				Desc:  li.Description,
				Qty:   li.Quantity,
				Price: li.UnitPrice,
			})
		}
	} else {
		data.Items = []lineItemJSON{{Desc: "", Qty: 1, Price: 0}}
	}

	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf(`{"items":[{"desc":"","qty":1,"price":0}],"taxRate":%f}`, taxRate)
	}
	return string(b)
}
