package models

import "errors"

var ErrAccountNotFound = errors.New("account not found")
var ErrAccountNotEnoughMoney = errors.New("not enough money")
