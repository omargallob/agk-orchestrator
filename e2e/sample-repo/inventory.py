"""In-memory widget inventory."""

# Current on-hand quantity for each widget SKU.
STOCK = {
    "blue-widget": 120,
    "green-widget": 0,
    "red-widget": 45,
}


def in_stock(sku):
    """Report whether the given SKU has any units on hand."""
    return STOCK.get(sku, 0) > 0
