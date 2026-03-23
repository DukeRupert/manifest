// Alpine.js component for invoice line items with running totals.
function invoiceForm(data) {
  return {
    items: data.items || [{ desc: '', qty: 1, price: 0 }],
    taxRate: data.taxRate || 0,

    addItem() {
      this.items.push({ desc: '', qty: 1, price: 0 });
    },

    removeItem(index) {
      if (this.items.length > 1) {
        this.items.splice(index, 1);
      }
    },

    subtotal() {
      return this.items.reduce(function (sum, item) {
        return sum + (item.qty * item.price || 0);
      }, 0);
    },

    taxAmount() {
      return this.subtotal() * (this.taxRate / 100);
    },

    total() {
      return this.subtotal() + this.taxAmount();
    }
  };
}
