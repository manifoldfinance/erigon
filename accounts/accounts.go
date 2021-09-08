// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package accounts implements high level Ethereum account management.
package accounts

import (
	"github.com/ledgerwatch/erigon/common"
)

const (
	// MimetypeDataWithValidator = "data/validator"
	// MimetypeTypedData         = "data/typed"
	// MimetypeClique            = "application/x-clique-header"
	MimetypeParlia            = "application/x-parlia-header"
	// MimetypeTextPlain         = "text/plain"
)

// Account represents an Ethereum account located at a specific location defined
// by the optional URL field.
type Account struct {
	Address common.Address `json:"address"` // Ethereum account address derived from the key
	// URL     URL            `json:"url"`     // Optional resource locator within a backend
}
