package models

import "errors"

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountNotEnoughMoney = errors.New("not enough money")
var ErrIdempotencyConflict = errors.New("idempotency key reused with different parameters")
