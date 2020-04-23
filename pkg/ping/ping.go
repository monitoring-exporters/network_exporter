package ping

import (
	"bytes"
	"fmt"
	"time"

	"github.com/syepes/ping_exporter/pkg/common"
	"github.com/syepes/ping_exporter/pkg/icmp"
)

// Ping ICMP Operation
func Ping(addr string, count int, interval time.Duration, timeout time.Duration) (*PingResult, error) {
	var out PingResult
	var err error

	pingOptions := &PingOptions{}
	pingOptions.SetCount(count)
	pingOptions.SetTimeout(timeout)
	pingOptions.SetInterval(interval)

	// Resolve hostnames
	ipAddrs, err := common.DestAddrs(addr)
	if err != nil || len(ipAddrs) == 0 {
		return nil, fmt.Errorf("Ping Failed due to an error: %v", err)
	}

	out = runPing(ipAddrs[0], pingOptions)

	return &out, nil
}

// PingString ICMP Operation
func PingString(addr string, count int, timeout time.Duration, interval time.Duration) (result string, err error) {
	pingOptions := &PingOptions{}
	pingOptions.SetCount(count)
	pingOptions.SetTimeout(timeout)
	pingOptions.SetInterval(interval)

	// Resolve hostnames
	ipAddrs, err := common.DestAddrs(addr)
	if err != nil || len(ipAddrs) == 0 {
		return result, err
	}

	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Start %v, PING %v (%v)\n", time.Now().Format("2006-01-02 15:04:05"), addr, ipAddrs[0]))
	begin := time.Now().UnixNano() / 1e6
	pingResult := runPing(ipAddrs[0], pingOptions)
	end := time.Now().UnixNano() / 1e6

	buffer.WriteString(fmt.Sprintf("%v packets transmitted, %v packet loss, time %vms\n", count, pingResult.DropRate, end-begin))
	buffer.WriteString(fmt.Sprintf("rtt min/avg/max = %v/%v/%v ms\n", common.Time2Float(pingResult.WorstTime), common.Time2Float(pingResult.AvgTime), common.Time2Float(pingResult.BestTime)))

	result = buffer.String()

	return result, nil
}

func runPing(ipAddr string, option *PingOptions) (pingResult PingResult) {
	pingResult.DestAddr = ipAddr

	pid := common.Goid()
	timeout := option.Timeout()
	interval := option.Interval()
	ttl := defaultTTL
	pingReturn := PingReturn{}

	seq := 0
	for cnt := 0; cnt < option.Count(); cnt++ {
		icmpReturn, err := icmp.Icmp(ipAddr, ttl, pid, timeout, seq)
		if err != nil || !icmpReturn.Success || !common.IsEqualIP(ipAddr, icmpReturn.Addr) {
			continue
		}

		pingReturn.allTime = append(pingReturn.allTime, icmpReturn.Elapsed)

		pingReturn.succSum++
		if pingReturn.worstTime == time.Duration(0) || icmpReturn.Elapsed > pingReturn.worstTime {
			pingReturn.worstTime = icmpReturn.Elapsed
		}
		if pingReturn.bestTime == time.Duration(0) || icmpReturn.Elapsed < pingReturn.bestTime {
			pingReturn.bestTime = icmpReturn.Elapsed
		}
		pingReturn.sumTime += icmpReturn.Elapsed
		pingReturn.avgTime = time.Duration((int64)(pingReturn.sumTime/time.Microsecond)/(int64)(pingReturn.succSum)) * time.Microsecond
		pingReturn.success = true

		seq++
		time.Sleep(interval)
	}

	if !pingReturn.success {
		pingResult.Success = false
		pingResult.DropRate = 100.0
		return pingResult
	}

	pingResult.Success = pingReturn.success
	pingResult.DropRate = float64(option.Count()-pingReturn.succSum) / float64(option.Count())
	pingResult.SumTime = pingReturn.sumTime
	pingResult.AvgTime = pingReturn.avgTime
	pingResult.BestTime = pingReturn.bestTime
	pingResult.WorstTime = pingReturn.worstTime
	pingResult.StdDev = common.StdDev(pingReturn.allTime, pingReturn.avgTime)

	return pingResult
}