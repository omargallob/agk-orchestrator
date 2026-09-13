"""Widget pricing."""

# BASE_PRICE_CENTS is the price of a single widget, in US cents.
# The end-to-end suite asserts that an agent can discover this value.
BASE_PRICE_CENTS = 1999


def total_price_cents(qty):
    """Return the price of ``qty`` widgets, in cents.

    There is no bulk discount: the total is the base price times the quantity.
    """
    if qty < 0:
        return 0
    return qty * BASE_PRICE_CENTS
