package tc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestmethod/inslice"
)

func ListKernelMods(verbose bool) (mods []string, err error) {
	logf(verbose, "(ListKernelMods) Running [lsmod]")
	out, err := exec.Command("lsmod").CombinedOutput()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		mods = append(mods, strings.Trim(strings.Split(strings.Split(strings.Trim(scanner.Text(), "\r\n\t "), " ")[0], "\t")[0], "\r\n\t "))
	}
	logf(verbose, "(ListKernelMods) return")
	return
}

func (f *FilterMatches) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}
	fmatch := &FilterMatch{}
	err := json.Unmarshal(data, fmatch)
	if err != nil {
		return err
	}
	*f = append(*f, fmatch)
	return nil
}

// no-json fallback for qdisc list
func qdiscListNoJson(verbose bool) ([]*Qdisc, error) {
	comm := []string{"tc", "qdisc", "show"}
	logf(verbose, "(qdiscListNoJson) Running %v", comm)
	out, err := exec.Command(comm[0], comm[1:]...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, string(out))
	}
	qdiscs := []*Qdisc{}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var qd *Qdisc
	for scanner.Scan() {
		line := scanner.Text()
		items := strings.Split(line, " ")
		if len(items) >= 2 && items[0] == "qdisc" {
			qd = &Qdisc{
				Kind: &items[1],
			}
			if len(items) >= 3 {
				qd.Handle = &items[2]
				if len(items) >= 5 && items[3] == "dev" {
					qd.Dev = &items[4]
					if len(items) >= 6 && items[5] == "root" {
						// root device
						isRoot := true
						qd.Root = &isRoot
						if len(items) >= 8 && items[6] == "refcnt" {
							refcnt, _ := strconv.Atoi(items[7])
							qd.Refcnt = &refcnt
							if len(items) >= 10 && items[8] == "bands" {
								bands, _ := strconv.Atoi(items[9])
								qd.Options = &QdiscOptions{
									PrioBands: &bands,
								}
								if len(items) >= 12 && items[10] == "priomap" {
									prios := []int{}
									for _, item := range items[11:] {
										if item == "" {
											continue
										}
										prio, _ := strconv.Atoi(item)
										prios = append(prios, prio)
									}
									qd.Options.PrioMap = &prios
								}
							}
						}
					} else if len(items) >= 7 && items[5] == "parent" {
						// netem device
						qd.Parent = &items[6]
						// parse limit, delay, loss, rate
						offset := 9
						for len(items) >= offset {
							if qd.Options == nil {
								qd.Options = &QdiscOptions{}
							}
							qdiscListNoJsonParseNetem(qd, items[offset-2], items[offset-1])
							offset = offset + 2
						}
					}
				}
			}
			qdiscs = append(qdiscs, qd)
		}
	}

	logf(verbose, "(qdiscListNoJson) return")
	return qdiscs, nil
}

func qdiscListNoJsonParseNetem(qd *Qdisc, name string, value string) {
	switch name {
	case "limit":
		lim, _ := strconv.Atoi(value)
		qd.Options.NetemLimit = &lim
	case "loss":
		loss, _ := strconv.Atoi(strings.TrimSuffix(value, "%"))
		lossFloat := float64(loss) / 100
		qd.Options.NetemLossRandom = &NetemLossRandom{
			Loss: lossFloat,
		}
	case "delay":
		multiplier := float64(1)
		if strings.HasSuffix(value, "us") {
			value = strings.TrimSuffix(value, "us")
			multiplier = 1000000
		} else if strings.HasSuffix(value, "ms") {
			multiplier = 1000
			value = strings.TrimSuffix(value, "ms")
		} else {
			value = strings.TrimSuffix(value, "s")
		}
		delay, _ := strconv.Atoi(value)
		multiplier = float64(delay) / multiplier
		qd.Options.NetemDelay = &NetemDelay{
			Delay: multiplier,
		}
	case "rate":
		multiplier := 1
		if strings.HasSuffix(value, "Gbit") {
			multiplier = 1024 * 1024 * 1024 / 8
			value = strings.TrimSuffix(value, "Gbit")
		} else if strings.HasSuffix(value, "Mbit") {
			multiplier = 1024 * 1024 / 8
			value = strings.TrimSuffix(value, "Mbit")
		} else if strings.HasSuffix(value, "Kbit") {
			multiplier = 1024 / 8
			value = strings.TrimSuffix(value, "Kbit")
		} else if strings.HasSuffix(value, "bit") {
			multiplier = 8
			value = strings.TrimSuffix(value, "bit")
		}
		rate, _ := strconv.Atoi(value)
		multiplier = rate * multiplier
		qd.Options.NetemRate = &NetemRate{
			Rate: multiplier,
		}
	}
}

// List tc qdisc
func ListQdisc(verbose bool) ([]*Qdisc, error) {
	comm := []string{"tc", "-j", "qdisc", "show"}
	logf(verbose, "(ListQdisc) Running %v", comm)
	out, err := exec.Command(comm[0], comm[1:]...).CombinedOutput()
	if err != nil {
		logf(verbose, "(ListQdisc) invoking tc with '-j' failed, failing back to old iproute2")
		defer logf(verbose, "(ListQdisc) return")
		return qdiscListNoJson(verbose)
	}
	qdiscs := []*Qdisc{}
	logf(verbose, "(ListQdisc) json.Unmarshal")
	err = json.Unmarshal(out, &qdiscs)
	if err != nil {
		logf(verbose, "(ListQdisc) invoking tc with '-j' failed on unmarshal, failing back to old iproute2")
		defer logf(verbose, "(ListQdisc) return")
		return qdiscListNoJson(verbose)
	}
	logf(verbose, "(ListQdisc) return")
	return qdiscs, nil
}

// no-json fallback for filter list
func filterListNoJson(iface string, verbose bool) ([]*Filter, error) {
	filters := []*Filter{}
	comm := []string{"tc", "filter", "show", "dev", iface}
	logf(verbose, "(filterListNoJson) Running %v", comm)
	out, err := exec.Command(comm[0], comm[1:]...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, string(out))
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var filter *Filter
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, " ") {
			if filter != nil {
				filters = append(filters, filter)
			}
			filter = &Filter{}
		}
		items := strings.Split(strings.Trim(line, "\r\n\t "), " ")
		if len(items) >= 3 && items[0] == "filter" && items[1] == "parent" {
			filter.Parent = &items[2]
			if len(items) >= 5 && items[3] == "protocol" {
				filter.Protocol = &items[4]
				if len(items) >= 7 && items[5] == "pref" {
					pref, _ := strconv.Atoi(items[6])
					filter.Pref = &pref
					if len(items) >= 8 {
						filter.Kind = &items[7]
						if len(items) >= 10 && items[8] == "chain" {
							chain, _ := strconv.Atoi(items[9])
							filter.Chain = &chain
							if len(items) >= 12 && items[10] == "fh" {
								filter.Options = &FilterOptions{
									FH: &items[11],
								}
								if len(items) >= 15 && items[12] == "ht" && items[13] == "divisor" {
									htDiv, _ := strconv.Atoi(items[14])
									filter.Options.HtDivisor = &htDiv
								} else if len(items) >= 14 && items[12] == "order" {
									order, _ := strconv.Atoi(items[13])
									filter.Options.Order = &order
									if len(items) >= 17 && items[14] == "key" && items[15] == "ht" {
										filter.Options.KeyHt = &items[16]
										if len(items) >= 19 && items[17] == "bkt" {
											filter.Options.Bkt = &items[18]
											if len(items) >= 21 && items[19] == "flowid" {
												filter.Options.FlowId = &items[20]
												if len(items) >= 22 && items[21] == "not_in_hw" {
													isTrue := true
													filter.Options.NotInHw = &isTrue
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		} else if len(items) == 4 && items[0] == "match" && items[2] == "at" {
			offset, _ := strconv.Atoi(items[3])
			valueMask := strings.Split(items[1], "/")
			value := valueMask[0]
			mask := ""
			if len(valueMask) > 1 {
				mask = valueMask[1]
			}
			filter.Options.Match = append(filter.Options.Match, &FilterMatch{
				Value:  value,
				Mask:   mask,
				Offset: offset,
			})
		}
	}
	if filter != nil {
		filters = append(filters, filter)
	}
	logf(verbose, "(filterListNoJson) return")
	return filters, nil
}

// List tc filters
func ListFilter(iface string, verbose bool) ([]*Filter, error) {
	filters := []*Filter{}
	comm := []string{"tc", "-j", "filter", "show", "dev", iface}
	logf(verbose, "(ListFilter) Running %v", comm)
	out, err := exec.Command(comm[0], comm[1:]...).CombinedOutput()
	if err != nil {
		logf(verbose, "(ListFilter) invoking tc with '-j' failed, failing back to old iproute2")
		filters, err = filterListNoJson(iface, verbose)
		if err != nil {
			return nil, err
		}
	} else {
		logf(verbose, "(ListFilter) json.Unmarshal")
		err = json.Unmarshal(out, &filters)
		if err != nil {
			logf(verbose, "(ListFilter) json failed - old iproute2 - attempting to fallback on string parsing")
			filters, err = filterListNoJson(iface, verbose)
			if err != nil {
				return nil, err
			}
		}
	}
	for i, filter := range filters {
		logf(verbose, "(ListFilter) Enum, filter=%d", i)
		filters[i].Iface = iface
		if filter.Options == nil {
			continue
		}
		for matchi, match := range filter.Options.Match {
			logf(verbose, "(ListFilter) Enum, filter=%d match=%d", i, matchi)
			for len(match.Value) < 8 {
				match.Value = "0" + match.Value
			}
			for len(match.Mask) < 8 {
				match.Mask = "0" + match.Mask
			}
			switch match.Offset {
			case 12: // source ip
				ip, err := parseOctets(match.Value)
				if err != nil {
					continue
				}
				maskInt, err := strconv.ParseUint(match.Mask, 16, 32)
				if err != nil {
					continue
				}
				mask := "/" + strconv.Itoa(bits.OnesCount(uint(maskInt)))
				if mask == "/32" {
					mask = ""
				}
				filters[i].Options.MatchParsed.SourceIPMask = StringToPtr(ip + mask)
			case 16: // dest ip
				ip, err := parseOctets(match.Value)
				if err != nil {
					continue
				}
				maskInt, err := strconv.ParseUint(match.Mask, 16, 32)
				if err != nil {
					continue
				}
				mask := "/" + strconv.Itoa(bits.OnesCount(uint(maskInt)))
				if mask == "/32" {
					mask = ""
				}
				filters[i].Options.MatchParsed.DestIPMask = StringToPtr(ip + mask)
			case 20: // ports
				if strings.HasPrefix(match.Mask, "ffff") {
					// parse source IP from first 2 bytes
					port, err := strconv.ParseUint(match.Value[0:4], 16, 16)
					if err != nil {
						continue
					}
					portInt := int(port)
					filters[i].Options.MatchParsed.SourcePort = &portInt
				}
				if strings.HasSuffix(match.Mask, "ffff") {
					// parse dest IP from last 2 bytes
					port, err := strconv.ParseUint(match.Value[4:8], 16, 16)
					if err != nil {
						continue
					}
					portInt := int(port)
					filters[i].Options.MatchParsed.DestPort = &portInt
				}
			}
		}
	}
	logf(verbose, "(ListFilter) return")
	return filters, nil
}

func parseOctets(val string) (string, error) {
	v, err := strconv.ParseUint(val[0:2], 16, 8)
	if err != nil {
		return "", err
	}
	if len(val) == 2 {
		return strconv.Itoa(int(v)), nil
	}
	n, err := parseOctets(val[2:])
	if err != nil {
		return "", err
	}
	return strconv.Itoa(int(v)) + "." + n, nil
}

// List interfaces
func ListIface() ([]string, error) {
	l, err := net.Interfaces()
	if err != nil {
		return nil, err

	}
	iface := []string{}
	for _, f := range l {
		if f.Name == "lo" {
			continue
		}
		iface = append(iface, f.Name)
	}
	return iface, nil
}

// Calls all the listing systems and combines them into a single full listing output
func ListRules(verbose bool) (*Rules, error) {
	logf(verbose, "(ListRules) ListIface")
	ifaces, err := ListIface()
	if err != nil {
		return nil, err
	}
	r := &Rules{
		Interfaces: ifaces,
	}
	for _, iface := range ifaces {
		logf(verbose, "(ListRules) ListFilter iface=%s", iface)
		filter, err := ListFilter(iface, verbose)
		if err != nil {
			return nil, err
		}
		r.Filters = append(r.Filters, filter...)
	}
	logf(verbose, "(ListRules) ListQdisc")
	qd, err := ListQdisc(verbose)
	if err != nil {
		return nil, err
	}
	r.Qdisc = qd
	// join qdisk and filter to make rules
	qdiscs := []int{}
	for fi, f := range r.Filters {
		logf(verbose, "(ListRules) Enum, filter=%d", fi)
		if f.Options == nil {
			continue
		}
		if f.Options.FlowId == nil {
			continue
		}
		for qi, q := range r.Qdisc {
			logf(verbose, "(ListRules) Enum, filter=%d qdisc=%d", fi, qi)
			if q.Kind == nil || *q.Kind != "netem" {
				continue
			}
			if q.Dev == nil || *q.Dev != f.Iface {
				continue
			}
			if q.Parent == nil || *q.Parent != *f.Options.FlowId {
				continue
			}
			if q.Options == nil {
				continue
			}
			var latency *string
			var linkSpeed *string
			var packetLoss *string
			if q.Options.NetemDelay != nil {
				latency = StringToPtr(fmt.Sprintf("%0.0f", q.Options.NetemDelay.Delay*1000))
			}
			if q.Options.NetemRate != nil {
				linkSpeed = StringToPtr(fmt.Sprintf("%d", q.Options.NetemRate.Rate))
			}
			if q.Options.NetemLossRandom != nil {
				packetLoss = StringToPtr(fmt.Sprintf("%0.2f", q.Options.NetemLossRandom.Loss*100))
			}
			if !inslice.HasInt(qdiscs, qi) {
				qdiscs = append(qdiscs, qi)
			}
			var sport *string
			var dport *string
			if f.Options.MatchParsed.SourcePort != nil {
				sport = StringToPtr(strconv.Itoa(*f.Options.MatchParsed.SourcePort))
			}
			if f.Options.MatchParsed.DestPort != nil {
				dport = StringToPtr(strconv.Itoa(*f.Options.MatchParsed.DestPort))
			}
			r.Rules = append(r.Rules, &Rule{
				Iface:              &f.Iface,
				SourceIP:           f.Options.MatchParsed.SourceIPMask,
				SourcePort:         sport,
				DestinationIP:      f.Options.MatchParsed.DestIPMask,
				DestinationPort:    dport,
				LatencyMs:          latency,
				PacketLossPct:      packetLoss,
				LinkSpeedRateBytes: linkSpeed,
				FlowID:             f.Options.FlowId,
				FilterNo:           fi,
				QdiscNo:            qi,
				FilterHandle:       f.Options.FH,
				QdiscHandle:        q.Handle,
			})
			break
		}
	}
	for qi, q := range qd {
		logf(verbose, "(ListRules) Enum, qdisc=%d", qi)
		if inslice.HasInt(qdiscs, qi) {
			continue
		}
		if q.Kind == nil || *q.Kind != "netem" {
			continue
		}
		if q.Dev == nil {
			continue
		}
		if q.Parent == nil {
			continue
		}
		if q.Options == nil {
			continue
		}
		var latency *string
		var linkSpeed *string
		var packetLoss *string
		if q.Options.NetemDelay != nil {
			latency = StringToPtr(fmt.Sprintf("%0.0f", q.Options.NetemDelay.Delay*1000))
		}
		if q.Options.NetemRate != nil {
			linkSpeed = StringToPtr(fmt.Sprintf("%d", q.Options.NetemRate.Rate))
		}
		if q.Options.NetemLossRandom != nil {
			packetLoss = StringToPtr(fmt.Sprintf("%0.2f", q.Options.NetemLossRandom.Loss*100))
		}
		r.Rules = append(r.Rules, &Rule{
			Iface:              q.Dev,
			LatencyMs:          latency,
			PacketLossPct:      packetLoss,
			LinkSpeedRateBytes: linkSpeed,
			FlowID:             q.Parent,
			QdiscNo:            qi,
			QdiscHandle:        q.Handle,
		})
	}
	logf(verbose, "(ListRules) return")
	return r, nil
}

func PtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func StringToPtr(str string) *string {
	return &str
}
