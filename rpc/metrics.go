// Copyright 2020 The go-ethereum Authors
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

package rpc

import (
	"fmt"

	victoriaMetrics "github.com/VictoriaMetrics/metrics"
	"github.com/ledgerwatch/erigon/metrics"
)

var (
	rpcRequestGauge    = victoriaMetrics.GetOrCreateCounter("rpc_total")
	failedReqeustGauge = victoriaMetrics.GetOrCreateCounter("rpc_failure")
	RpcServingTimer    = metrics.NewRegisteredTimer("rpc/duration/all", nil)
)

func newRPCServingTimerMS(method string, valid bool) *victoriaMetrics.Summary {
	flag := "success"
	if !valid {
		flag = "failure"
	}
	m := fmt.Sprintf(`rpc_duration_seconds{method="%s",success="%s"}`, method, flag)
	return victoriaMetrics.GetOrCreateSummary(m)
}

func newRPCRequestGauge(method string) metrics.Gauge {
	m := fmt.Sprintf("rpc/count/%s", method)
	return metrics.GetOrRegisterGauge(m, nil)
}