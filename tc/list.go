package tc

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestmethod/inslice"
)

/*
=-=-=-=-= filter:lo =-=-=-=-=
[]
=-=-=-=-= filter:enp0s5 =-=-=-=-=
[
  {
    "parent": "1:",
    "protocol": "ip",
    "pref": 3,
    "kind": "u32",
    "chain": 0,
    "options": null
  },
  {
    "parent": "1:",
    "protocol": "ip",
    "pref": 3,
    "kind": "u32",
    "chain": 0,
    "options": {
      "fh": "800:",
      "ht_divisor": 1,
      "order": null,
      "key_ht": null,
      "bkt": null,
      "flowid": null,
      "not_in_hw": null,
      "match": null
    }
  },
  {
    "parent": "1:",
    "protocol": "ip",
    "pref": 3,
    "kind": "u32",
    "chain": 0,
    "options": {
      "fh": "800::800",
      "ht_divisor": null,
      "order": 2048,
      "key_ht": "800",
      "bkt": "0",
      "flowid": "1:3",
      "not_in_hw": true,
      "match": [
        {
          "value": "c0a80001",
          "mask": "ffffffff",
          "offmask": "",
          "off": 12
        },
        {
          "value": "c0a80032",
          "mask": "ffffffff",
          "offmask": "",
          "off": 16
        },
        {
          "value": "bb80bb8",
          "mask": "ffffffff",
          "offmask": "",
          "off": 20
        }
      ]
    }
  }
]
=-=-=-=-= qdisc =-=-=-=-=
[
  {
    "kind": "noqueue",
    "handle": "0:",
    "dev": "lo",
    "root": true,
    "refcnt": 2,
    "parent": null,
    "options": {
      "bands": null,
      "priomap": null,
      "multiqueue": null,
      "limit": null,
      "delay": null,
      "loss-random": null,
      "rate": null,
      "ecn": null,
      "gap": null
    }
  },
  {
    "kind": "prio",
    "handle": "1:",
    "dev": "enp0s5",
    "root": true,
    "refcnt": 2,
    "parent": null,
    "options": {
      "bands": 3,
      "priomap": [
        1,
        2,
        2,
        2,
        1,
        2,
        0,
        0,
        1,
        1,
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "multiqueue": false,
      "limit": null,
      "delay": null,
      "loss-random": null,
      "rate": null,
      "ecn": null,
      "gap": null
    }
  },
  {
    "kind": "netem",
    "handle": "30:",
    "dev": "enp0s5",
    "root": null,
    "refcnt": null,
    "parent": "1:3",
    "options": {
      "bands": null,
      "priomap": null,
      "multiqueue": null,
      "limit": 1000,
      "delay": {
        "delay": 0.1,
        "jitter": 0,
        "correlation": 0
      },
      "loss-random": {
        "loss": 0.1,
        "correlation": 0
      },
      "rate": {
        "rate": 100000,
        "packetoverhead": 0,
        "cellsize": 0,
        "celloverhead": 0
      },
      "ecn": false,
      "gap": 0
    }
  }
]
*/

type Qdisc struct {
	Kind    *string `json:"kind"`
	Handle  *string `json:"handle"`
	Dev     *string `json:"dev"`
	Root    *bool   `json:"root"`
	Refcnt  *int    `json:"refcnt"`
	Parent  *string `json:"parent"`
	Options *struct {
		PrioBands      *int   `json:"bands"`
		PrioMap        *[]int `json:"priomap"`
		PrioMultiqueue *bool  `json:"multiqueue"`
		NetemLimit     *int   `json:"limit"`
		NetemDelay     *struct {
			Delay       float64 `json:"delay"`
			Jitter      float64 `json:"jitter"`
			Correlation float64 `json:"correlation"`
		} `json:"delay"`
		NetemLossRandom *struct {
			Loss        float64 `json:"loss"`
			Correlation float64 `json:"correlation"`
		} `json:"loss-random"`
		NetemRate *struct {
			Rate           int `json:"rate"`
			PacketOverhead int `json:"packetoverhead"`
			CellSize       int `json:"cellsize"`
			CellOverhead   int `json:"celloverhead"`
		} `json:"rate"`
		NetemEcn *bool    `json:"ecn"`
		NetemGap *float64 `json:"gap"`
	} `json:"options"`
}

type Filter struct {
	Iface    string  `json:"interface"`
	Parent   *string `json:"parent"`
	Protocol *string `json:"protocol"`
	Pref     *int    `json:"pref"`
	Kind     *string `json:"kind"`
	Chain    *int    `json:"chain"`
	Options  *struct {
		FH          *string       `json:"fh"`
		HtDivisor   *int          `json:"ht_divisor"`
		Order       *int          `json:"order"`
		KeyHt       *string       `json:"key_ht"`
		Bkt         *string       `json:"bkt"`
		FlowId      *string       `json:"flowid"`
		NotInHw     *bool         `json:"not_in_hw"`
		Match       FilterMatches `json:"match"`
		MatchParsed struct {
			SourceIPMask *string `json:"source_ip_mask"`
			DestIPMask   *string `json:"dest_ip_mask"`
			SourcePort   *int    `json:"source_port"`
			DestPort     *int    `json:"dest_port"`
		} `json:"match_parsed"`
	} `json:"options"`
}

type FilterMatch struct {
	Value   string `json:"value"`
	Mask    string `json:"mask"`
	Offmask string `json:"offmask"`
	Offset  int    `json:"off"`
}

type FilterMatches []*FilterMatch

type Rules struct {
	Interfaces []string
	Qdisc      []*Qdisc
	Filters    []*Filter
	Rules      []*Rule
}

type Rule struct {
	// set, delete
	Iface           *string
	SourceIP        *string
	SourcePort      *string
	DestinationIP   *string
	DestinationPort *string
	// set only
	LatencyMs          *string
	PacketLossPct      *string
	LinkSpeedRateBytes *string
	// output only parameters
	FlowID       *string
	FilterNo     int
	FilterHandle *string
	QdiscNo      int
	QdiscHandle  *string
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

// List tc qdisc
func ListQdisc() ([]*Qdisc, error) {
	out, err := exec.Command("tc", "-j", "qdisc", "show").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, string(out))
	}
	qdiscs := []*Qdisc{}
	err = json.Unmarshal(out, &qdiscs)
	if err != nil {
		return nil, err
	}
	return qdiscs, nil
}

// List tc filters
func ListFilter(iface string) ([]*Filter, error) {
	out, err := exec.Command("tc", "-j", "filter", "show", "dev", iface).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, string(out))
	}
	filters := []*Filter{}
	err = json.Unmarshal(out, &filters)
	if err != nil {
		return nil, err
	}
	for i, filter := range filters {
		filters[i].Iface = iface
		if filter.Options == nil {
			continue
		}
		for _, match := range filter.Options.Match {
			for len(match.Value) < 8 {
				match.Value = "0" + match.Value
			}
			for len(match.Mask) < 8 {
				match.Value = "0" + match.Value
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
		iface = append(iface, f.Name)
	}
	return iface, nil
}

// Calls all the listing systems and combines them into a single full listing output
func ListRules() (*Rules, error) {
	ifaces, err := ListIface()
	if err != nil {
		return nil, err
	}
	r := &Rules{
		Interfaces: ifaces,
	}
	for _, iface := range ifaces {
		filter, err := ListFilter(iface)
		if err != nil {
			return nil, err
		}
		r.Filters = append(r.Filters, filter...)
	}
	qd, err := ListQdisc()
	if err != nil {
		return nil, err
	}
	r.Qdisc = qd
	// join qdisk and filter to make rules
	qdiscs := []int{}
	for fi, f := range r.Filters {
		if f.Options == nil {
			continue
		}
		if f.Options.FlowId == nil {
			continue
		}
		for qi, q := range r.Qdisc {
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
